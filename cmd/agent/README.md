# cmd/agent

Агент отправляет собранные метрики пакетами сервису gRPC. Адрес сервера настраивается флагами `-g` или `-grpc-address`, переменной `GRPC_ADDRESS` либо полем `grpc_address` в JSON конфигурации. По умолчанию используется `localhost:3200`. Для проверки `trusted_subnet` агент передаёт свой адрес в gRPC metadata `x-real-ip`.
