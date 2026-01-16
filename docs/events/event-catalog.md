# Catálogo de Eventos

## Estructura Base de Eventos

Todos los eventos comparten la siguiente estructura:

```json
{
  "id": "uuid",
  "type": "domain.event_name",
  "occurred_at": "2026-01-15T10:00:00Z",
  "payment_id": "pay-123",
  "user_id": "user-456",
  "amount": 100.5,
  "currency": "USD",
  "reason": "string opcional",
  "reservation_id": "res-789",
  "gateway_ref": "GW-ABC123"
}
```

---

## Eventos de Payment

### payment.initiated

Emitido cuando se crea un nuevo pago.

| Campo      | Tipo    | Descripción            |
| ---------- | ------- | ---------------------- |
| payment_id | string  | ID único del pago      |
| user_id    | string  | ID del usuario         |
| amount     | decimal | Monto del pago         |
| currency   | string  | Moneda (USD, MXN, EUR) |

**Productor:** payment-orchestrator  
**Consumidores:** wallet-service, metrics-collector

```json
{
  "id": "evt-001",
  "type": "payment.initiated",
  "occurred_at": "2026-01-15T10:00:00Z",
  "payment_id": "pay-123",
  "user_id": "user-456",
  "amount": 100.5,
  "currency": "USD"
}
```

---

### payment.completed

Emitido cuando un pago se completa exitosamente.

| Campo       | Tipo    | Descripción            |
| ----------- | ------- | ---------------------- |
| payment_id  | string  | ID del pago            |
| user_id     | string  | ID del usuario         |
| amount      | decimal | Monto del pago         |
| gateway_ref | string  | Referencia del gateway |

**Productor:** payment-orchestrator  
**Consumidores:** metrics-collector

---

### payment.failed

Emitido cuando un pago falla en cualquier etapa.

| Campo      | Tipo   | Descripción      |
| ---------- | ------ | ---------------- |
| payment_id | string | ID del pago      |
| user_id    | string | ID del usuario   |
| reason     | string | Motivo del fallo |

**Productor:** payment-orchestrator  
**Consumidores:** metrics-collector, error-handler

---

## Eventos de Wallet

### wallet.funds_reserved

Emitido cuando se reservan fondos exitosamente.

| Campo          | Tipo    | Descripción          |
| -------------- | ------- | -------------------- |
| payment_id     | string  | ID del pago asociado |
| user_id        | string  | ID del usuario       |
| amount         | decimal | Monto reservado      |
| reservation_id | string  | ID de la reservación |

**Productor:** wallet-service  
**Consumidores:** gateway-processor, metrics-collector

```json
{
  "id": "evt-002",
  "type": "wallet.funds_reserved",
  "occurred_at": "2026-01-15T10:00:05Z",
  "payment_id": "pay-123",
  "user_id": "user-456",
  "amount": 100.5,
  "currency": "USD",
  "reservation_id": "res-789"
}
```

---

### wallet.reservation_failed

Emitido cuando no se pueden reservar fondos.

| Campo      | Tipo    | Descripción                                   |
| ---------- | ------- | --------------------------------------------- |
| payment_id | string  | ID del pago                                   |
| user_id    | string  | ID del usuario                                |
| amount     | decimal | Monto solicitado                              |
| reason     | string  | Motivo (insufficient funds, wallet not found) |

**Productor:** wallet-service  
**Consumidores:** payment-orchestrator, metrics-collector

---

### wallet.funds_deducted

Emitido cuando se confirma la deducción de fondos.

| Campo          | Tipo    | Descripción          |
| -------------- | ------- | -------------------- |
| payment_id     | string  | ID del pago          |
| user_id        | string  | ID del usuario       |
| reservation_id | string  | ID de la reservación |
| amount         | decimal | Monto deducido       |

**Productor:** wallet-service  
**Consumidores:** payment-orchestrator, metrics-collector

---

### wallet.funds_released

Emitido cuando se liberan fondos reservados.

| Campo          | Tipo   | Descripción          |
| -------------- | ------ | -------------------- |
| payment_id     | string | ID del pago          |
| reservation_id | string | ID de la reservación |
| reason         | string | Motivo de liberación |

**Productor:** wallet-service  
**Consumidores:** metrics-collector

---

## Eventos de Gateway

### gateway.payment_approved

Emitido cuando el gateway externo aprueba el pago.

| Campo          | Tipo   | Descripción            |
| -------------- | ------ | ---------------------- |
| payment_id     | string | ID del pago            |
| user_id        | string | ID del usuario         |
| reservation_id | string | ID de la reservación   |
| gateway_ref    | string | Referencia del gateway |

**Productor:** gateway-processor  
**Consumidores:** wallet-service, metrics-collector

```json
{
  "id": "evt-003",
  "type": "gateway.payment_approved",
  "occurred_at": "2026-01-15T10:00:10Z",
  "payment_id": "pay-123",
  "user_id": "user-456",
  "reservation_id": "res-789",
  "gateway_ref": "GW-ABC123"
}
```

---

### gateway.payment_rejected

Emitido cuando el gateway externo rechaza el pago.

| Campo          | Tipo   | Descripción          |
| -------------- | ------ | -------------------- |
| payment_id     | string | ID del pago          |
| user_id        | string | ID del usuario       |
| reservation_id | string | ID de la reservación |
| reason         | string | Motivo del rechazo   |

**Productor:** gateway-processor  
**Consumidores:** wallet-service, metrics-collector

---

## Flujo de Eventos - Happy Path

```
1. payment.initiated      (orchestrator → wallet)
2. wallet.funds_reserved  (wallet → gateway)
3. gateway.payment_approved (gateway → wallet)
4. wallet.funds_deducted  (wallet → orchestrator)
5. payment.completed      (orchestrator → metrics)
```

## Flujo de Eventos - Insufficient Funds

```
1. payment.initiated           (orchestrator → wallet)
2. wallet.reservation_failed   (wallet → orchestrator)
3. payment.failed              (orchestrator → metrics)
```

## Flujo de Eventos - Gateway Rejected

```
1. payment.initiated          (orchestrator → wallet)
2. wallet.funds_reserved      (wallet → gateway)
3. gateway.payment_rejected   (gateway → wallet)
4. wallet.funds_released      (wallet → orchestrator)
5. payment.failed             (orchestrator → metrics)
```

---

## Topología de Colas SQS

| Cola              | Productor             | Consumidor        |
| ----------------- | --------------------- | ----------------- |
| wallet-queue      | orchestrator, gateway | wallet-service    |
| gateway-queue     | wallet-service        | gateway-processor |
| wallet-queue-dlq  | SQS (auto)            | error-handler     |
| gateway-queue-dlq | SQS (auto)            | error-handler     |

## EventBridge

| Bus            | Patrón | Destino           |
| -------------- | ------ | ----------------- |
| payment-events | \*     | metrics-collector |

---

# Esquema de Base de Datos

## DynamoDB Tables

### payments-table

| Atributo    | Tipo   | Key |
| ----------- | ------ | --- |
| id          | String | PK  |
| user_id     | String | GSI |
| service_id  | String | -   |
| amount      | Number | -   |
| currency    | String | -   |
| status      | String | -   |
| description | String | -   |
| created_at  | String | -   |
| updated_at  | String | -   |

**GSI:** user_id-index (user_id → id)

---

### wallets-table

| Atributo   | Tipo   | Key |
| ---------- | ------ | --- |
| id         | String | PK  |
| user_id    | String | GSI |
| balance    | String | -   |
| currency   | String | -   |
| updated_at | String | -   |
| version    | Number | -   |

**GSI:** user_id-index (user_id → id)

**Nota:** `version` se usa para optimistic locking.

---

### reservations-table

| Atributo   | Tipo   | Key |
| ---------- | ------ | --- |
| id         | String | PK  |
| payment_id | String | GSI |
| user_id    | String | -   |
| amount     | String | -   |
| currency   | String | -   |
| status     | String | -   |
| expires_at | String | -   |
| created_at | String | -   |

**GSI:** payment_id-index (payment_id → id)

**Estados:** active, confirmed, released

---

### failed-events-table

| Atributo       | Tipo   | Key |
| -------------- | ------ | --- |
| id             | String | PK  |
| original_event | String | -   |
| event_type     | String | GSI |
| payment_id     | String | -   |
| error_message  | String | -   |
| source         | String | -   |
| retry_count    | Number | -   |
| status         | String | -   |
| created_at     | String | -   |

**GSI:** event_type-index (event_type → id)

---

## Capacidad y Escalamiento

### Modo On-Demand

Todas las tablas usan **On-Demand capacity** para escalar automáticamente según la demanda.

### TTL

- `reservations-table`: TTL en `expires_at` para limpiar reservaciones expiradas automáticamente.

### Consistencia

- Lecturas: Eventually consistent (default)
- Escrituras: Conditional writes con `version` para optimistic locking
