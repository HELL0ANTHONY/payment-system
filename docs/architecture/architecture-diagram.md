# Arquitectura del Sistema de Pagos

## Diagrama de Alto Nivel

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                                   CLIENTES                                      │
└─────────────────────────────────────┬───────────────────────────────────────────┘
                                      │
                                      ▼
┌─────────────────────────────────────────────────────────────────────────────────┐
│                               API GATEWAY                                       │
│                         POST /payments | GET /payments/{id}                     │
└─────────────────────────────────────┬───────────────────────────────────────────┘
                                      │
                                      ▼
┌─────────────────────────────────────────────────────────────────────────────────┐
│                          PAYMENT-ORCHESTRATOR                                   │
│                              (Lambda)                                           │
│  • Validación de requests                                                       │
│  • Creación de pagos                                                            │
│  • Consulta de estado                                                           │
├─────────────────────────────────────────────────────────────────────────────────┤
│  DynamoDB: payments-table                                                       │
└──────────────────┬──────────────────────────────────────┬───────────────────────┘
                   │                                      │
                   │ SQS: wallet-queue                    │ EventBridge
                   │ (PaymentInitiated)                   │ (todos los eventos)
                   ▼                                      ▼
┌─────────────────────────────────────┐    ┌─────────────────────────────────────┐
│          WALLET-SERVICE             │    │        METRICS-COLLECTOR            │
│             (Lambda)                │    │            (Lambda)                 │
│  • Reservar fondos                  │    │  • Registrar métricas               │
│  • Confirmar deducción              │    │  • Contadores de eventos            │
│  • Liberar fondos                   │    │  • Montos por tipo                  │
├─────────────────────────────────────┤    ├─────────────────────────────────────┤
│  DynamoDB: wallets-table            │    │  CloudWatch Metrics                 │
│  DynamoDB: reservations-table       │    │                                     │
└──────────────────┬──────────────────┘    └─────────────────────────────────────┘
                   │
                   │ SQS: gateway-queue
                   │ (FundsReserved)
                   ▼
┌─────────────────────────────────────┐
│        GATEWAY-PROCESSOR            │
│            (Lambda)                 │
│  • Integración con pasarela         │
│  • Manejo de respuestas             │
│  • Reintentos                       │
├─────────────────────────────────────┤
│  External Gateway (Mock)            │
└──────────────────┬──────────────────┘
                   │
                   │ SQS: wallet-queue
                   │ (GatewayApproved/Rejected)
                   ▼
          ┌───────┴───────┐
          │               │
          ▼               ▼
    ┌──────────┐    ┌──────────┐
    │ Approved │    │ Rejected │
    │ConfirmDed│    │ReleaseFun│
    └──────────┘    └──────────┘


                    DEAD LETTER QUEUES
                          │
                          ▼
┌─────────────────────────────────────┐
│          ERROR-HANDLER              │
│            (Lambda)                 │
│  • Procesar mensajes fallidos       │
│  • Reintentos automáticos           │
│  • Almacenar para revisión          │
├─────────────────────────────────────┤
│  DynamoDB: failed-events-table      │
└─────────────────────────────────────┘
```

## Flujo de Eventos - Happy Path

```
┌────────┐     ┌─────────────┐     ┌────────────┐     ┌─────────────┐     ┌────────────┐
│ Client │     │ Orchestrator│     │   Wallet   │     │   Gateway   │     │   Wallet   │
└───┬────┘     └──────┬──────┘     └─────┬──────┘     └──────┬──────┘     └─────┬──────┘
    │                 │                  │                   │                  │
    │ POST /payments  │                  │                   │                  │
    │────────────────>│                  │                   │                  │
    │                 │                  │                   │                  │
    │                 │ PaymentInitiated │                   │                  │
    │                 │─────────────────>│                   │                  │
    │                 │                  │                   │                  │
    │                 │                  │ FundsReserved     │                  │
    │                 │                  │──────────────────>│                  │
    │                 │                  │                   │                  │
    │                 │                  │                   │ ProcessPayment   │
    │                 │                  │                   │──────┐           │
    │                 │                  │                   │      │           │
    │                 │                  │                   │<─────┘           │
    │                 │                  │                   │                  │
    │                 │                  │                   │ GatewayApproved  │
    │                 │                  │                   │─────────────────>│
    │                 │                  │                   │                  │
    │                 │                  │                   │                  │ ConfirmDeduction
    │                 │                  │                   │                  │──────┐
    │                 │                  │                   │                  │      │
    │                 │                  │                   │                  │<─────┘
    │                 │                  │                   │                  │
    │ 202 Accepted    │                  │                   │                  │
    │<────────────────│                  │                   │                  │
    │                 │                  │                   │                  │
```

## Flujo de Error - Saldo Insuficiente

```
┌────────┐     ┌─────────────┐     ┌────────────┐
│ Client │     │ Orchestrator│     │   Wallet   │
└───┬────┘     └──────┬──────┘     └─────┬──────┘
    │                 │                  │
    │ POST /payments  │                  │
    │────────────────>│                  │
    │                 │                  │
    │                 │ PaymentInitiated │
    │                 │─────────────────>│
    │                 │                  │
    │                 │                  │ Check Balance
    │                 │                  │──────┐
    │                 │                  │      │ balance < amount
    │                 │                  │<─────┘
    │                 │                  │
    │                 │ ReservationFailed│
    │                 │<─────────────────│
    │                 │                  │
    │ 202 Accepted    │                  │
    │<────────────────│                  │
    │                 │                  │
    │ (Estado: failed)│                  │
```

## Componentes AWS

| Componente    | Servicio AWS | Propósito                          |
| ------------- | ------------ | ---------------------------------- |
| API           | API Gateway  | Endpoint HTTP REST                 |
| Compute       | Lambda       | Procesamiento serverless           |
| Mensajería    | SQS          | Comunicación async entre servicios |
| Eventos       | EventBridge  | Fan-out de eventos para métricas   |
| Base de datos | DynamoDB     | Almacenamiento de estado           |
| Métricas      | CloudWatch   | Observabilidad                     |
