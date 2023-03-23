# Morrigan

OpenAI/ChatGPT 后端

## APIs

### Get welcome message and new conversation ID

<details>
 <summary><code>GET</code> <code><b>/api/welcome</b></code></summary>

##### Parameters

> None

##### Responses

> | http code     | content-type                      | response                                           |
> |---------------|-----------------------------------|---------------------------------------------------------------------|
> | `200`         | `application/json`        | `{"message": "welcome message", "id": "new-cid"}`                                         |


</details>

### Post chat prompt and return Server Side Event

<details>
 <summary><code>GET</code> <code><b>/api/chat-sse</b></code></summary>

##### Parameters

> | name              |  type     | data type      | description                         |
> |-------------------|-----------|----------------|-------------------------------------|
> | `csid` |  optional | string    | conversation ID        |
> | `prompt` |  required | string   | message for ask        |


##### Responses

> | http code     | content-type                      | response                                           |
> |---------------|-----------------------------------|---------------------------------------------------------------------|
> | `200`         | `text/event-stream`        | `{"delta": "message fragments", "id": "conversation ID"}`                                         |


</details>

## Getting started

```bash

test -e .env || cp .env.example .env
# edit .env and change api key of OpenAI

make deps

forego start

# or

make dist GOMOD=auto


```
