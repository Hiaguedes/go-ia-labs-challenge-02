# go ia labs challenge 02

## Objetivo

Sistema distribuído em Go com dois microsserviços (Serviço A e Serviço B) que cooperam para consultar o clima de uma cidade a partir de um CEP. A observabilidade é feita com OpenTelemetry (OTEL): os traços das requisições são coletados pelo OTEL Collector e visualizados no Zipkin (e também no Jaeger).

### Arquitetura

1. **Serviço A (`service_cep`, porta 8000)**: recebe o CEP por `POST`, valida e encaminha para o Serviço B. Devolve a resposta do Serviço B.
2. **Serviço B (`service_weather`, porta 8001)**: recebe o CEP, identifica a cidade (ViaCEP), consulta a temperatura (WeatherAPI) e faz as conversões.
3. **OTEL Collector + Zipkin + Jaeger**: coleta e visualização dos traços. O Prometheus raspa as métricas e o Grafana as exibe.

```
cliente / driver ──POST──▶ Serviço A ──GET──▶ Serviço B ──▶ ViaCEP / WeatherAPI
                              │                  │
                              └──── spans ───────┴──▶ OTEL Collector ──▶ Zipkin / Jaeger
```

Cada requisição carrega um `X-Request-ID`, que é repassado do Serviço A ao Serviço B, registrado nos logs e anexado como atributo `request.id` nos spans.

## Configuração

1. Crie o `.env` a partir do exemplo:

   ```bash
   cp .env.example .env
   ```

2. Preencha `WEATHER_API_KEY` com a sua chave do [WeatherAPI](https://www.weatherapi.com/).
3. Ajuste as portas dos serviços se necessário (`WEBSERVER_CEP_PORT`, `WEBSERVER_WEATHER_PORT`).

## Subindo o ecossistema

Com Docker instalado, na raiz do projeto:

```bash
docker compose up --build -d
```

Isso sobe os dois serviços e a infraestrutura de observabilidade:

| Componente   | URL                          | Uso                                   |
|--------------|------------------------------|---------------------------------------|
| Serviço A    | http://localhost:8000        | Requisição POST com o CEP             |
| Serviço B    | http://localhost:8001        | Consulta direta (GET `?cep=`)         |
| **Zipkin**   | **http://localhost:9411**    | Visualização dos traços               |
| Jaeger       | http://localhost:16686       | Visualização alternativa dos traços   |
| Prometheus   | http://localhost:9090        | Métricas                              |
| Grafana      | http://localhost:3001        | Dashboards (usuário/senha padrão: admin/admin) |

Para parar tudo: `docker compose down`.

## Requisição POST no Serviço A

O endpoint recebe um JSON com o campo `cep`:

```bash
curl -i -X POST http://localhost:8000/ \
  -H 'Content-Type: application/json' \
  -d '{"cep": "29193530"}'
```

Também é possível usar o arquivo [`api/cep.http`](api/cep.http) no VS Code (extensão REST Client) ou no IntelliJ.

Resposta de sucesso (`200 OK`):

```json
{"city":"Aracruz","temp_C":"18.30","temp_F":"65.00","temp_K":"291.45"}
```

Erros:

| Status | Corpo                  | Quando                                      |
|--------|------------------------|---------------------------------------------|
| 422    | `invalid zipcode`      | CEP com formato inválido ou corpo malformado |
| 404    | `can not find zipcode` | CEP válido, mas não encontrado              |
| 502    | `Error fetching weather data` | Falha ao consultar o Serviço B      |

O CEP pode ser enviado com ou sem hífen (`29193-530`).

## Visualizando os traços no Zipkin

1. Gere algumas requisições (POST no Serviço A, ou o driver de métricas, descrito abaixo).
2. Abra **http://localhost:9411** e clique em **Run Query** (ou *Find a trace*).
3. Em **serviceName**, escolha:
   - `service-cep`: traços que passaram pelo Serviço A (inclui a chamada ao Serviço B);
   - `metrics-client`: traços iniciados pelo driver, com o fluxo completo.
4. Clique em um traço para abrir a árvore de spans:

   ```
   HTTP POST [service-cep]            ← requisição recebida pelo Serviço A
   └── weather.request [service-cep]  ← chamada do Serviço A ao Serviço B
       └── HTTP GET [service-weather] ← requisição recebida pelo Serviço B
   ```

   Cada span mostra a duração e as tags `http.response.status_code`, `url.full` e `request.id`.

### Filtrando por request id

Para encontrar a requisição de um log específico:

- Nos logs dos serviços aparece `request_id=<valor>`:

  ```bash
  docker compose logs service_cep | grep request_id
  ```

- No Zipkin, no campo **annotationQuery** (ou pela API), use `request.id=<valor>`:

  ```bash
  curl -G "http://localhost:9411/api/v2/traces" --data-urlencode "annotationQuery=request.id=<valor>"
  ```

## Métricas

- **Prometheus** (http://localhost:9090): targets em *Status → Targets*. Os serviços expõem `/metrics`. Métricas úteis: `http_server_request_duration_seconds` (latência por rota e status) e `cep_weather_flow_duration_seconds` (fluxo completo, enviado pelo driver).
- **Grafana** (http://localhost:3001): adicione um datasource do tipo Prometheus com a URL `http://prometheus:9090`.

## Driver de métricas e traços (opcional)

O comando abaixo executa o fluxo a partir de um cliente Go, que gera spans e métricas:

```bash
make metrics-run CEP=29193530 RUNS=3
```

Antes, `make metrics-up` garante que a infraestrutura de observabilidade está no ar.

## Executando sem Docker

Em terminais separados (é preciso um `.env` na raiz e a infraestrutura de observabilidade rodando com `make metrics-up`):

```bash
make cep-run       # Serviço A em :8000
make weather-run   # Serviço B em :8001
```

Localmente, os serviços enviam os spans para `localhost:4317`. No compose, a variável `OTEL_EXPORTER_OTLP_ENDPOINT` aponta para `otel-collector:4317`.

## Estrutura

- `cmd/cep`, `cmd/weather`: entrypoints dos serviços A e B.
- `internal/`: handlers, casos de uso e regras de negócio.
- `configs/`: carregamento de configuração (viper).
- `metrics/`: driver de métricas e traços, configuração do OTEL Collector e do Prometheus.
- `api/`: arquivos `.http` com exemplos de requisição.
