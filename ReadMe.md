# go ia labs challenge 02

## Objetivo

Desenvolver um sistema distribuído em Go composto por dois microsserviços (Serviço A e Serviço B) que cooperam para consultar o clima de uma cidade baseada no CEP. O diferencial deste desafio é a implementação de Observabilidade utilizando OpenTelemetry (OTEL) e Zipkin para realizar o rastreamento distribuído (Distributed Tracing) das requisições.

### Arquitetura do sistema

O sistema é composto por:

1. **Serviço A (Input) service-cep**: Recebe a requisição do usuário, valida o CEP e encaminha para o Serviço B
2. **Serviço B (Orquestração) service-weather**: Recebe o CEP, identifica a cidade, consulta a temperatura e realiza as conversões.
3. **OTEL + Zipkin**: Infraestrutura de coleta e visualização dos traços.