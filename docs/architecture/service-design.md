# Documento de Diseño de Servicios

## Visión General

El sistema de pagos está compuesto por 5 microservicios implementados como AWS Lambda, comunicados mediante eventos a través de SQS y EventBridge.

---

## 1. Payment Orchestrator

### Responsabilidad

Punto de entrada del sistema. Gestiona el ciclo de vida de los pagos.

### Límites del Servicio

- Recibe requests HTTP de clientes
- Crea y persiste pagos
- Inicia el flujo de eventos
- Consulta estado de pagos

### API

| Método | Ruta           | Descripción      |
| ------ | -------------- | ---------------- |
| POST   | /payments      | Crear nuevo pago |
| GET    | /payments/{id} | Consultar estado |

### Request - Crear Pago

```json
{
  "user_id": "user-123",
  "service_id": "service-456",
  "amount": 100.5,
  "currency": "USD",
  "description": "Pago de servicio"
}
```

### Response

```json
{
  "success": true,
  "data": {
    "id": "pay-789",
    "user_id": "user-123",
    "status": "pending",
    "amount": "100.50",
    "currency": "USD",
    "created_at": "2026-01-15T10:00:00Z"
  }
}
```

### Dependencias

- **DynamoDB**: payments-table
- **SQS**: wallet-queue (publica)
- **EventBridge**: payment-events (publica)

### Estados del Pago

```
pending → reserved → processing → completed
    │         │           │
    └─────────┴───────────┴──→ failed
```

---

## 2. Wallet Service

### Responsabilidad

Gestiona los saldos de usuarios y las reservaciones de fondos.

### Límites del Servicio

- Verifica disponibilidad de fondos
- Crea reservaciones temporales (TTL: 15 min)
- Confirma deducciones
- Libera fondos en caso de fallo

### Eventos que Consume

| Evento                   | Acción              |
| ------------------------ | ------------------- |
| payment.initiated        | Reservar fondos     |
| gateway.payment_approved | Confirmar deducción |
| gateway.payment_rejected | Liberar reservación |

### Eventos que Produce

| Evento                    | Condición              |
| ------------------------- | ---------------------- |
| wallet.funds_reserved     | Reserva exitosa        |
| wallet.reservation_failed | Sin fondos suficientes |
| wallet.funds_deducted     | Deducción confirmada   |
| wallet.funds_released     | Reserva liberada       |

### Dependencias

- **DynamoDB**: wallets-table, reservations-table
- **SQS**: wallet-queue (consume), gateway-queue (publica)

### Modelo de Datos

**Wallet**

```json
{
  "id": "wallet-123",
  "user_id": "user-456",
  "balance": "1000.00",
  "currency": "USD",
  "version": 1
}
```

**Reservation**

```json
{
  "id": "res-789",
  "payment_id": "pay-123",
  "user_id": "user-456",
  "amount": "100.00",
  "status": "active|confirmed|released",
  "expires_at": "2026-01-15T10:15:00Z"
}
```

### Concurrencia

Utiliza **optimistic locking** con campo `version` para prevenir race conditions en actualizaciones de balance.

---

## 3. Gateway Processor

### Responsabilidad

Integración con pasarelas de pago externas.

### Límites del Servicio

- Recibe solicitudes de procesamiento
- Comunica con gateway externo
- Maneja respuestas y errores
- Simula latencia y fallos (mock)

### Eventos que Consume

| Evento                | Acción                   |
| --------------------- | ------------------------ |
| wallet.funds_reserved | Procesar pago en gateway |

### Eventos que Produce

| Evento                   | Condición       |
| ------------------------ | --------------- |
| gateway.payment_approved | Gateway aprueba |
| gateway.payment_rejected | Gateway rechaza |

### Dependencias

- **SQS**: gateway-queue (consume), wallet-queue (publica)
- **External**: Payment Gateway API (mock)

### Configuración del Mock

```go
MockGateway{
  FailRate: 0.1  // 10% de fallos simulados
}
```

---

## 4. Metrics Collector

### Responsabilidad

Recolección y registro de métricas del sistema.

### Límites del Servicio

- Escucha todos los eventos del sistema
- Registra métricas en CloudWatch
- No modifica estado ni produce eventos

### Eventos que Consume

Todos los eventos vía EventBridge.

### Métricas Registradas

| Métrica        | Tipo    | Dimensiones         |
| -------------- | ------- | ------------------- |
| EventCount     | Counter | EventType           |
| PaymentAmount  | Gauge   | EventType, Currency |
| PaymentSuccess | Counter | -                   |
| PaymentFailure | Counter | FailureType         |

### Dependencias

- **EventBridge**: payment-events (consume)
- **CloudWatch**: PaymentSystem namespace

---

## 5. Error Handler

### Responsabilidad

Procesamiento de mensajes fallidos y recuperación.

### Límites del Servicio

- Consume de Dead Letter Queues
- Decide si reintentar o almacenar
- Registra fallos para análisis

### Lógica de Reintento

```
if retryCount < maxRetries AND isRetryable(event):
    → Reenviar a cola original
else:
    → Almacenar en failed-events-table
```

### Eventos Retryables

- `payment.initiated`
- `wallet.funds_reserved`

### Eventos No Retryables

- `payment.completed`
- `gateway.payment_approved`
- `gateway.payment_rejected`

### Dependencias

- **SQS**: \*-dlq (consume), wallet-queue (publica)
- **DynamoDB**: failed-events-table

### Modelo de Datos

**FailedEvent**

```json
{
  "id": "msg-123",
  "original_event": "{...}",
  "event_type": "payment.initiated",
  "payment_id": "pay-456",
  "error_message": "max retries exceeded",
  "source": "wallet-queue-dlq",
  "retry_count": 3,
  "status": "failed",
  "created_at": "2026-01-15T10:00:00Z"
}
```

---

## Patrones de Comunicación

### Síncrono

- Cliente → API Gateway → Payment Orchestrator

### Asíncrono (Coreografía)

- Payment Orchestrator → SQS → Wallet Service
- Wallet Service → SQS → Gateway Processor
- Gateway Processor → SQS → Wallet Service

### Fan-out

- Todos los servicios → EventBridge → Metrics Collector

### Error Handling

- Todas las colas → DLQ → Error Handler

---

## Variables de Entorno por Servicio

### payment-orchestrator

```
PAYMENTS_TABLE=payments
WALLET_QUEUE_URL=https://sqs.../wallet-queue
```

### wallet-service

```
WALLETS_TABLE=wallets
RESERVATIONS_TABLE=reservations
GATEWAY_QUEUE_URL=https://sqs.../gateway-queue
```

### gateway-processor

```
WALLET_QUEUE_URL=https://sqs.../wallet-queue
```

### metrics-collector

```
METRICS_NAMESPACE=PaymentSystem
```

### error-handler

```
FAILED_EVENTS_TABLE=failed-events
WALLET_QUEUE_URL=https://sqs.../wallet-queue
MAX_RETRIES=3
```
