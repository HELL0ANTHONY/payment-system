# Sistema de Pagos Orientado a Eventos

Sistema de procesamiento de pagos basado en arquitectura event-driven usando AWS Lambda, SQS, EventBridge y DynamoDB.

## Arquitectura

```
Cliente → API Gateway → Payment Orchestrator
                              │
                              ▼ (PaymentInitiated)
                        Wallet Service
                              │
                              ▼ (FundsReserved)
                       Gateway Processor
                              │
                    ┌─────────┴─────────┐
                    ▼                   ▼
              (Approved)           (Rejected)
                    │                   │
                    └─────────┬─────────┘
                              ▼
                        Wallet Service
                              │
                              ▼
                    Metrics / Error Handler
```

## Servicios

| Servicio             | Trigger     | Responsabilidad                  |
| -------------------- | ----------- | -------------------------------- |
| payment-orchestrator | API Gateway | Crear y consultar pagos          |
| wallet-service       | SQS         | Gestionar saldos y reservaciones |
| gateway-processor    | SQS         | Integración con pasarela externa |
| metrics-collector    | EventBridge | Registrar métricas en CloudWatch |
| error-handler        | SQS DLQ     | Reintentos y manejo de fallos    |

## Stack Tecnológico

| Componente    | Tecnología  | Justificación                           |
| ------------- | ----------- | --------------------------------------- |
| Lenguaje      | Go 1.25     | Rendimiento, cold starts bajos (~100ms) |
| Compute       | AWS Lambda  | Serverless, escalamiento automático     |
| API           | API Gateway | REST, throttling, auth integrado        |
| Mensajería    | SQS         | Garantía de entrega, DLQ nativo         |
| Eventos       | EventBridge | Fan-out, filtros, bajo acoplamiento     |
| Base de datos | DynamoDB    | Serverless, escalamiento on-demand      |
| Métricas      | CloudWatch  | Nativo AWS, dashboards, alarmas         |

## Estructura del Proyecto

```
payment-system/
├── shared/                    # Código compartido
│   ├── events/               # Definición de eventos
│   └── publisher/            # Cliente SQS
├── lambdas/
│   ├── payment-orchestrator/ # API de pagos
│   ├── wallet-service/       # Gestión de billetera
│   ├── gateway-processor/    # Integración gateway
│   ├── metrics-collector/    # Métricas CloudWatch
│   └── error-handler/        # Manejo de errores
├── docs/
│   ├── architecture/         # Diagramas y diseño
│   └── events/              # Catálogo de eventos
├── go.work                   # Go workspaces
├── Makefile                  # Comandos de build
└── README.md
```

## Requisitos

- Go 1.25+
- AWS CLI configurado
- Make

## Ejecutar Tests

```bash
# Todos los tests
make test

# Lambda específica
make test-payment-orchestrator
make test-wallet-service

# Con coverage
make coverage
```

## Build

```bash
# Todas las lambdas
make build

# Lambda específica
make build-payment-orchestrator
```

## Eventos del Sistema

| Evento                    | Productor    | Consumidor   |
| ------------------------- | ------------ | ------------ |
| payment.initiated         | orchestrator | wallet       |
| payment.completed         | orchestrator | metrics      |
| payment.failed            | orchestrator | metrics      |
| wallet.funds_reserved     | wallet       | gateway      |
| wallet.reservation_failed | wallet       | orchestrator |
| wallet.funds_deducted     | wallet       | orchestrator |
| wallet.funds_released     | wallet       | metrics      |
| gateway.payment_approved  | gateway      | wallet       |
| gateway.payment_rejected  | gateway      | wallet       |

## Manejo de Errores

### Estrategia de Reintentos

- Mensajes fallidos van a DLQ automáticamente
- error-handler procesa DLQ con política de reintentos
- Máximo 3 reintentos para eventos retryables
- Eventos no recuperables se almacenan para revisión manual

### Eventos Retryables

- `payment.initiated`
- `wallet.funds_reserved`

### Transacciones Compensatorias

- Si gateway rechaza → wallet libera fondos
- Si timeout → reservación expira (TTL 15 min)

## Concurrencia

Se usa **optimistic locking** en wallet-service:

- Campo `version` en wallets-table
- Condition expression en updates
- Retry automático en conflictos

## Observabilidad

### Logs

- Structured logging con `log/slog`
- Campos: payment_id, user_id, event_type

### Métricas CloudWatch

- EventCount por tipo de evento
- PaymentAmount por moneda
- PaymentSuccess / PaymentFailure

## Documentación

- [Diagrama de Arquitectura](docs/architecture/architecture-diagram.md)
- [Diseño de Servicios](docs/architecture/service-design.md)
- [Catálogo de Eventos](docs/events/event-catalog.md)

## Decisiones de Diseño

### ¿Por qué Coreografía vs Orquestación?

- Menor acoplamiento entre servicios
- Cada servicio es autónomo
- Mejor escalabilidad
- Más resiliente a fallos

### ¿Por qué DynamoDB?

- Serverless (no hay que gestionar infraestructura)
- Escalamiento on-demand
- Single-digit millisecond latency
- TTL nativo para expiración de reservaciones

### ¿Por qué SQS sobre SNS?

- Garantía de entrega at-least-once
- DLQ integrado
- Mejor para procesamiento secuencial
- Retry automático

## Autor

Jorge Fernández
