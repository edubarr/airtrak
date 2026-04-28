# AirTrak MVP Architecture and Implementation Plan

**Project:** AirTrak  
**Goal:** Open-source platform for managing airsoft games, initially WhatsApp-first.  
**Initial interfaces:** WhatsApp bot + full backend API.  
**Future interfaces:** React web app, Flutter app, Telegram bot, optional public REST API.  
**Primary backend language:** Go.  
**Primary database:** PostgreSQL.  
**Primary API style:** Connect RPC + Protobuf.  
**WhatsApp integration:** `whatsmeow`.  
**Natural language parsing:** generic LLM provider abstraction, initially OpenAI-compatible.  
**Languages:** `pt-BR` and `en`, configurable by group owner.

---

## 1. Product scope

### 1.1 MVP objective

The first useful version should allow a WhatsApp group to manage airsoft games without a web frontend.

Basic flow:

1. Add the bot to a WhatsApp group.
2. Group owner runs `/setup`.
3. Owner sets the group language.
4. Admin creates fields and games using fixed commands or natural language.
5. Players join or leave games using fixed commands or natural language.
6. Admins can list players and cancel games.
7. All operations are also available through a backend API for testing and future web/app integration.

### 1.2 Core MVP features

- WhatsApp group setup.
- Owner/admin/member roles.
- Fixed commands in Portuguese and English.
- Natural language parsing through LLM into structured intents.
- Confirmation flow for risky operations.
- i18n replies in `pt-BR` and `en`.
- Full Connect RPC API for all domain operations.
- PostgreSQL persistence.
- Outbox-based outgoing WhatsApp messages.
- Parser/debug API for testing LLM and command parsing without using WhatsApp.

### 1.3 Explicitly out of scope for the first MVP

- Payments.
- Public marketplace of fields.
- Ranking/scoring system.
- React frontend.
- Flutter app.
- Telegram bot.
- Official WhatsApp Business Cloud API.
- Complex team balancing.
- Waivers/signatures.

---

## 2. Architecture principles

1. **Modular monolith first.**  
   Keep deployment simple while organizing the code by modules.

2. **Domain services are the center.**  
   WhatsApp, API, future web, future mobile and future Telegram must all call the same application/domain services.

3. **WhatsApp is only an adapter.**  
   The WhatsApp bot receives messages, parses them, resolves intent and calls application services. It should not contain business rules directly.

4. **LLM is only a parser.**  
   The LLM must never execute actions. It only returns structured JSON that your Go code validates.

5. **Fixed commands remain first-class.**  
   Natural language can fail. Fixed commands are cheaper, deterministic, testable and reliable.

6. **Confirmation for destructive or high-impact actions.**  
   Creating games, cancelling games and leaving games should use confirmation when coming from natural language.

7. **All outgoing messages go through an outbox.**  
   This makes retries reliable and avoids losing notifications when WhatsApp is temporarily disconnected.

8. **Connect RPC is the primary API.**  
   Use Protobuf contracts for the API, generate Go and TypeScript clients later, and optionally add REST only for public third-party integrations.

---

## 3. High-level system architecture

```mermaid
flowchart TD
    WA[WhatsApp Group / DM] --> WM[whatsmeow Client]

    WM --> Ingest[Message Ingestion]
    Ingest --> Idem[Idempotency Check]
    Idem --> Ctx[Message Context Builder]
    Ctx --> InputRouter[Input Router]

    InputRouter --> Fixed[Fixed Command Parser]
    InputRouter --> LLMParser[LLM Natural Language Parser]

    Fixed --> Intent[Parsed Intent]
    LLMParser --> Intent

    Intent --> Resolver[Intent Resolver]
    Resolver --> Conv[Conversation State / Pending Action]
    Resolver --> App[Application Services]

    API[Connect RPC API] --> App
    Tests[API Tests / CLI / Future Web] --> API

    App --> Perm[Permission Layer]
    App --> DB[(PostgreSQL App DB)]
    App --> Outbox[Outgoing Message Outbox]
    App --> Audit[Audit Events]

    Worker[Worker] --> Outbox
    Worker --> WM

    WM --> WAStore[(whatsmeow SQL Store)]
    Admin[Admin HTTP / Bot Status] --> WM
    Admin --> DB
```

---

## 4. Runtime applications

The project should initially have four Go binaries.

```text
apps/
  bot/        WhatsApp event receiver and input processor
  api/        Connect RPC API for all operations
  worker/     outbox sender, reminders and async jobs
  admin/      QR/status/admin-only HTTP endpoints, optional if merged into api
```

### 4.1 `bot`

Responsibilities:

- Connect to WhatsApp through `whatsmeow`.
- Maintain one active WhatsApp session.
- Receive message events.
- Ignore irrelevant group messages.
- Persist incoming message IDs for idempotency.
- Build parsing context.
- Call fixed parser or LLM parser.
- Pass structured intent to the resolver.
- Queue replies in `outgoing_messages`.

The bot should not directly own business logic like capacity rules, permissions or game creation.

### 4.2 `api`

Responsibilities:

- Expose all application operations through Connect RPC.
- Expose parser/debug operations for development.
- Expose future web/mobile-compatible API.
- Authenticate callers.
- Authorize operations through the same permission layer used by WhatsApp.

### 4.3 `worker`

Responsibilities:

- Send pending outgoing WhatsApp messages.
- Retry failed sends.
- Process future scheduled jobs, such as reminders.
- Process async notifications.

### 4.4 `admin`

Responsibilities:

- Health checks.
- WhatsApp status.
- QR/pairing status.
- Manual reconnect.
- Operational diagnostics.

For the MVP, `admin` can be merged into `api` if easier, but the endpoints should remain clearly separated and protected.

---

## 5. Internal module layout

Recommended repository layout:

```text
airtrak/
  README.md
  LICENSE
  docker-compose.yml
  .env.example
  Makefile
  AGENTS.md

  backend/
    go.mod

    apps/
      bot/
      api/
      worker/
      admin/

    internal/
      whatsapp/
        client.go
        events.go
        qr.go
        sender.go

      input/
        router.go
        fixed_parser.go
        llm_parser.go
        intent.go
        resolver.go
        context_builder.go

      llm/
        provider.go
        openai_provider.go
        schema.go
        prompt.go
        mock_provider.go

      i18n/
        localizer.go
        locales/
          pt-BR.toml
          en.toml

      auth/
      groups/
      fields/
      games/
      registrations/
      permissions/
      conversations/
      outbox/
      audit/

      db/
        migrations/
        queries/

      platform/
        config/
        logger/
        clock/
        ids/

  proto/
    buf.yaml
    buf.gen.yaml
    airtrak/
      auth/v1/auth.proto
      groups/v1/groups.proto
      fields/v1/fields.proto
      games/v1/games.proto
      registrations/v1/registrations.proto
      parser/v1/parser.proto
      bot/v1/bot.proto

  docs/
    architecture.md
    commands.md
    api.md
```

---

## 6. API architecture

### 6.1 Recommendation

Use **Connect RPC + Protobuf** as the primary API.

Reasoning:

- Works well with Go.
- Gives strongly typed contracts.
- Supports generated TypeScript clients for a future React frontend.
- Can support generated Dart clients for a future Flutter app.
- Allows schema linting and breaking-change checks with Buf.
- Can still be called over HTTP with JSON, which helps testing.
- Avoids maintaining separate REST DTOs and Protobuf DTOs from day one.

### 6.2 API adapter model

```mermaid
flowchart TD
    Core[Application Services / Domain Logic]

    WhatsApp[WhatsApp Adapter] --> Core
    Connect[Connect RPC API Adapter] --> Core
    FutureWeb[Future React App] --> Connect
    FutureFlutter[Future Flutter App] --> Connect
    OptionalREST[Optional REST Facade Later] --> Core

    Core --> DB[(PostgreSQL)]
    Core --> Outbox[Outbox]
```

### 6.3 API service boundaries

Recommended Protobuf service packages:

```text
airtrak.auth.v1
airtrak.groups.v1
airtrak.fields.v1
airtrak.games.v1
airtrak.registrations.v1
airtrak.parser.v1
airtrak.bot.v1
```

Recommended services:

| Service | Purpose |
|---|---|
| `AuthService` | API login/session later; admin token in MVP. |
| `GroupService` | Setup group, group config, members, roles. |
| `FieldService` | Create/list/update/delete fields and aliases. |
| `GameService` | Create/list/get/cancel games. |
| `RegistrationService` | Join/leave games and list registrations. |
| `ParserService` | Parse text and preview resolved intents for tests/debugging. |
| `BotService` | WhatsApp status, QR, reconnect and operational endpoints. |

### 6.4 API method map

#### `GroupService`

- `SetupGroup`
- `GetGroup`
- `ListGroups`
- `UpdateGroupConfig`
- `SetGroupLocale`
- `AddAdmin`
- `RemoveAdmin`
- `ListMembers`

#### `FieldService`

- `CreateField`
- `ListFields`
- `UpdateField`
- `DeleteField`

#### `GameService`

- `CreateGame`
- `ListGames`
- `GetGame`
- `CancelGame`
- `ListPlayers`

#### `RegistrationService`

- `JoinGame`
- `LeaveGame`
- `ChangeSlotType`
- `GetMyRegistrations`

#### `ParserService`

- `ParseInput`
- `PreviewResolvedIntent`
- `ConfirmPendingAction`
- `CancelPendingAction`
- `GetConversationState`

#### `BotService`

- `GetWhatsappStatus`
- `GetPairingQRCode`
- `ReconnectWhatsapp`

---

## 7. API design settings

### 7.1 Connect RPC settings

Recommended settings:

```text
API_PROTOCOL=connect
API_ENABLE_GRPC_WEB=true
API_ENABLE_GRPC=true
API_ENABLE_REFLECTION=true in development, false or protected in production
API_ENABLE_HEALTH=true
API_JSON_TRANSCODING=false initially
API_MAX_REQUEST_BODY_BYTES=1048576
API_REQUEST_TIMEOUT_SECONDS=15
API_SERVER_READ_TIMEOUT_SECONDS=10
API_SERVER_WRITE_TIMEOUT_SECONDS=30
API_SERVER_IDLE_TIMEOUT_SECONDS=120
```

### 7.2 CORS settings

For MVP development:

```text
API_CORS_ALLOWED_ORIGINS=http://localhost:5173,http://localhost:3000
API_CORS_ALLOWED_METHODS=GET,POST,OPTIONS
API_CORS_ALLOWED_HEADERS=Authorization,Content-Type,Connect-Protocol-Version,Connect-Timeout-Ms,X-Requested-With
API_CORS_EXPOSED_HEADERS=Grpc-Status,Grpc-Message,Grpc-Status-Details-Bin
API_CORS_ALLOW_CREDENTIALS=true
```

For production, never use wildcard origins if cookies or credentials are enabled.

### 7.3 API auth phases

#### Phase 1: development/internal API

```text
API_AUTH_MODE=admin_token
API_ADMIN_TOKEN=<random-long-token>
```

Use this for testing and initial Codex-driven development.

#### Phase 2: real user API

Add:

- `users`
- `sessions`
- email/password or magic link login
- account-to-WhatsApp identity linking

#### Phase 3: external API

Add:

- OAuth2 client credentials or API keys
- scoped tokens
- rate limits by client

---

## 8. WhatsApp architecture

### 8.1 Important note about `whatsmeow`

`whatsmeow` is a Go library for WhatsApp Web multidevice. It is not the official WhatsApp Business Cloud API. This is fine for an open-source community MVP, but for official business-grade usage the project should later support the official WhatsApp Business API as a separate adapter.

### 8.2 WhatsApp process model

Run only one active bot instance per WhatsApp session.

Recommended settings:

```text
WA_ENABLED=true
WA_DATABASE_URL=postgres://whatsmeow:whatsmeow@postgres-wa:5432/whatsmeow?sslmode=disable
WA_CONNECT_TIMEOUT_SECONDS=30
WA_RECONNECT_ENABLED=true
WA_RECONNECT_MIN_SECONDS=5
WA_RECONNECT_MAX_SECONDS=300
WA_SEND_RATE_LIMIT_PER_SECOND=1
WA_SEND_MAX_RETRIES=5
WA_IGNORE_NON_COMMAND_GROUP_MESSAGES=true
WA_PROCESS_ONLY_WHEN_MENTIONED=true
WA_BOT_MENTION_ALIASES=airtrak,bot
WA_SINGLE_INSTANCE_LOCK=true
```

### 8.3 Message processing rules

In WhatsApp groups, process a message only if:

1. it starts with `/`; or
2. it mentions the bot; or
3. it replies to a bot message; or
4. the sender has an active conversation state.

This prevents the bot from parsing normal group conversation.

### 8.4 WhatsApp flow

```mermaid
sequenceDiagram
    participant User as WhatsApp User
    participant WA as WhatsApp
    participant Bot as whatsmeow Bot
    participant Parser as Input Parser
    participant Resolver as Intent Resolver
    participant Service as App Service
    participant DB as PostgreSQL
    participant Worker as Worker

    User->>WA: @bot entrar no jogo da Spartacus
    WA->>Bot: Message event
    Bot->>DB: Save incoming message id
    Bot->>Parser: Parse fixed command or natural language
    Parser-->>Bot: ParsedIntent
    Bot->>Resolver: Resolve game/slot/ambiguity
    Resolver->>Service: JoinGame
    Service->>DB: Create registration
    Service->>DB: Queue outgoing reply
    Worker->>DB: Fetch pending message
    Worker->>WA: Send reply
    WA->>User: You joined the game
```

---

## 9. Input parsing architecture

### 9.1 Parser pipeline

```mermaid
flowchart TD
    Msg[Incoming Message] --> State[Conversation State Parser]
    State -->|handled| Intent[Parsed Intent]
    State -->|not handled| Fixed[Fixed Command Parser]
    Fixed -->|matched| Intent
    Fixed -->|not matched| LLM[LLM Parser]
    LLM --> Schema[Schema Validation]
    Schema --> Intent
    Intent --> Resolver[Intent Resolver]
    Resolver --> Execute[Execute or Ask Follow-up]
```

Pipeline order:

1. Conversation state parser.
2. Fixed command parser.
3. LLM natural language parser.
4. Help fallback.

### 9.2 Fixed commands

Support Portuguese and English aliases.

#### Setup and config

```text
/setup
/config idioma pt-BR
/config language en
```

#### Games

```text
/jogos
/games

/jogo criar Spartacus | domingo 08:00 | aluguel 10 | aeg 10
/game create Spartacus | Sunday 08:00 | rental 10 | own_aeg 10

/jogo cancelar SPA-26-08
/game cancel SPA-26-08
```

#### Registration

```text
/entrar SPA-26-08
/join SPA-26-08

/sair SPA-26-08
/leave SPA-26-08

/jogadores SPA-26-08
/players SPA-26-08
```

### 9.3 Natural language examples

#### `pt-BR`

```text
@bot crie novo jogo na Spartacus domingo às 8 da manhã, com 10 vagas de aluguel e 10 vagas para AEG própria
@bot entrar no jogo da Spartacus
@bot sair do jogo de domingo
@bot quais são os próximos jogos?
```

#### `en`

```text
@bot create a new game at Spartacus on Sunday at 8am, with 10 rental slots and 10 own AEG slots
@bot join the Spartacus game
@bot leave Sunday's game
@bot show upcoming games
```

---

## 10. Parsed intent model

All input methods should produce the same internal object shape.

Conceptual structure:

```text
ParsedIntent
  intent: create_game | join_game | leave_game | list_games | list_players | cancel_game | help | unknown
  language: pt-BR | en
  confidence: 0.0 - 1.0
  source: conversation_state | fixed_command | llm | fallback
  entities:
    game_code
    field_name
    date_text
    slot_type
    slots[]
  missing_fields[]
  ambiguities[]
  needs_confirmation
```

The LLM returns a structured version of this. The Go resolver converts it into domain commands.

---

## 11. LLM architecture

### 11.1 Provider abstraction

The application should depend on an interface, not on OpenAI directly.

```text
LLMProvider
  ParseIntent(request) -> ParseResult
```

Initial providers:

```text
openai
mock
```

Future providers:

```text
anthropic
gemini
ollama
openrouter
local-model
```

### 11.2 Recommended LLM settings

```text
LLM_ENABLED=true
LLM_PROVIDER=openai
LLM_MODEL=gpt-4o-mini or another cost-effective structured-output-capable model
LLM_TEMPERATURE=0
LLM_TOP_P=1
LLM_MAX_OUTPUT_TOKENS=700
LLM_TIMEOUT_SECONDS=8
LLM_MAX_RETRIES=1
LLM_MIN_CONFIDENCE=0.65
LLM_FORCE_CONFIRMATION_FOR_CREATE=true
LLM_FORCE_CONFIRMATION_FOR_CANCEL=true
LLM_FORCE_CONFIRMATION_FOR_LEAVE=true
LLM_LOG_RAW_MESSAGES=false
LLM_LOG_PARSED_PAYLOAD=true
LLM_PROMPT_VERSION=airtrak-intent-v1
```

### 11.3 LLM prompt structure

Use a mostly static prompt for better reliability and caching.

```text
System/static section:
  - You are an intent parser for an airsoft game management WhatsApp bot.
  - You only return JSON matching the schema.
  - You never execute actions.
  - You never invent fields, games or users.
  - Supported languages: pt-BR and en.
  - Supported intents and entity definitions.
  - Examples in pt-BR.
  - Examples in en.

Variable section:
  - current time
  - timezone
  - group locale
  - known fields
  - upcoming games
  - sender registrations
  - user message
```

### 11.4 LLM output schema rules

The schema should require:

- intent enum
- language enum
- confidence number
- entities object
- missing fields list
- ambiguities list
- needs confirmation boolean

The LLM must not return database IDs unless those IDs were explicitly provided in context. It can return matched names or game codes.

### 11.5 LLM safety rules

The LLM must not:

- create records;
- update records;
- decide permissions;
- send WhatsApp messages;
- call tools directly;
- invent fields;
- invent games;
- finalize datetime without Go-side validation.

The Go resolver owns:

- date resolution;
- field lookup;
- game lookup;
- ambiguity resolution;
- permission checks;
- capacity checks;
- confirmations;
- database transactions.

---

## 12. Intent resolver

The resolver turns a parsed intent into a safe application command.

```mermaid
flowchart TD
    PI[Parsed Intent] --> Confidence{Confidence >= min?}
    Confidence -->|No| Low[Ask user to rephrase / show help]
    Confidence -->|Yes| Missing{Missing fields?}
    Missing -->|Yes| Ask[Ask follow-up question]
    Missing -->|No| Lookup[Lookup fields/games/slots]
    Lookup --> Ambiguous{Ambiguous?}
    Ambiguous -->|Yes| Choice[Ask user to choose]
    Ambiguous -->|No| Risky{Needs confirmation?}
    Risky -->|Yes| Pending[Create pending action]
    Risky -->|No| Exec[Execute domain service]
```

### 12.1 Resolver behavior

| Condition | Behavior |
|---|---|
| Low confidence | Ask user to rephrase and show examples. |
| Missing date for game creation | Ask for date/time. |
| Unknown field | Ask whether to create/use typed field name, or ask admin to create field first. |
| Multiple matching games | Ask user to select from numbered list. |
| Missing slot type | Ask user to choose rental or own AEG. |
| Create game via natural language | Always confirm. |
| Cancel game | Always confirm. |
| Leave game via natural language | Confirm unless exact game code was used. |
| Join game with one clear match and one slot type | Can execute directly. |

---

## 13. Conversation state and pending actions

Use conversation state for multi-step flows.

Examples:

```text
state:select_game
state:select_slot_type
state:awaiting_date
state:confirm_create_game
state:confirm_cancel_game
state:confirm_leave_game
```

Use pending actions for confirmed operations.

Examples:

```text
pending_action:create_game
pending_action:cancel_game
pending_action:leave_game
```

Recommended settings:

```text
CONVERSATION_STATE_TTL_MINUTES=15
PENDING_ACTION_TTL_MINUTES=15
PENDING_ACTION_MAX_PER_USER=5
```

Localized confirmation words:

```text
pt-BR yes: sim, s, confirmar, pode criar, pode cancelar
pt-BR no: não, nao, n, cancelar, deixa

en yes: yes, y, confirm, create it, cancel it
en no: no, n, cancel, nevermind
```

---

## 14. i18n architecture

### 14.1 Locale support

Initial supported locales:

```text
pt-BR
en
```

Owner can configure group language:

```text
/config idioma pt-BR
/config language en
```

Locale resolution order:

1. Sender-specific locale, if configured later.
2. Group locale.
3. App default locale.

Recommended settings:

```text
APP_DEFAULT_LOCALE=pt-BR
APP_SUPPORTED_LOCALES=pt-BR,en
APP_DEFAULT_TIMEZONE=America/Maceio
```

### 14.2 Translation files

```text
internal/i18n/locales/
  pt-BR.toml
  en.toml
```

Recommended message keys:

```text
help.general
setup.created
setup.already_exists
config.locale_changed
permission.denied
field.created
field.not_found
game.confirm_create
game.created
game.not_found
game.ambiguous
game.cancel_confirm
game.cancelled
game.list_empty
game.list_item
registration.joined
registration.left
registration.already_joined
registration.not_registered
slot.select_type
parser.low_confidence
parser.missing_date
parser.unknown
error.internal
```

---

## 15. Data model

### 15.1 Database split

Use two PostgreSQL databases or two schemas.

Recommended MVP:

```text
postgres-app       AirTrak application data
postgres-whatsmeow whatsmeow session data
```

This avoids accidental migration conflicts with `whatsmeow` tables.

### 15.2 App tables

#### `whatsapp_identities`

Represents a WhatsApp sender.

Important fields:

```text
id
jid
phone
push_name
locale
created_at
updated_at
```

#### `groups`

Represents an AirTrak group linked to a WhatsApp group.

Important fields:

```text
id
whatsapp_chat_jid
name
locale
owner_identity_id
created_at
updated_at
```

#### `group_members`

Important fields:

```text
group_id
identity_id
role: owner | admin | member
joined_at
```

#### `fields`

Important fields:

```text
id
group_id
name
aliases[]
created_at
updated_at
```

#### `games`

Important fields:

```text
id
group_id
field_id
code
title
starts_at
timezone
status: scheduled | cancelled | finished
created_by_identity_id
created_at
updated_at
```

#### `game_slot_types`

Important fields:

```text
id
game_id
type: rental | own_aeg
label
capacity
created_at
```

#### `game_registrations`

Important fields:

```text
id
game_id
identity_id
slot_type_id
status: registered | waitlisted | cancelled
registered_at
cancelled_at
```

#### `incoming_messages`

Used for idempotency.

Important fields:

```text
whatsapp_message_id
chat_jid
sender_jid
message_text
processed_at
```

#### `outgoing_messages`

Used for reliable sending.

Important fields:

```text
id
chat_jid
reply_to_message_id
text
status: pending | sending | sent | failed
attempts
last_error
created_at
sent_at
```

#### `pending_actions`

Important fields:

```text
id
chat_jid
sender_jid
action_type
payload
status: pending | confirmed | cancelled | expired
expires_at
created_at
```

#### `conversation_states`

Important fields:

```text
id
chat_jid
sender_jid
state_type
payload
expires_at
created_at
```

#### `llm_parse_logs`

Important fields:

```text
id
chat_jid
sender_jid
provider
model
locale
original_text nullable/configurable
parsed_intent
confidence
parsed_payload
input_tokens
output_tokens
latency_ms
status: success | failed | skipped
error
created_at
```

#### `audit_events`

Important fields:

```text
id
actor_identity_id
actor_user_id nullable later
group_id
entity_type
entity_id
action
metadata
created_at
```

---

## 16. Permission model

### 16.1 Roles

```text
owner
admin
member
```

### 16.2 Permissions

| Operation | Owner | Admin | Member |
|---|---:|---:|---:|
| Setup group | First setup only | No | No |
| Change group language | Yes | No initially | No |
| Add admin | Yes | No | No |
| Remove admin | Yes | No | No |
| Create field | Yes | Yes | No |
| Create game | Yes | Yes | No |
| Cancel game | Yes | Yes | No |
| List games | Yes | Yes | Yes |
| Join game | Yes | Yes | Yes |
| Leave own registration | Yes | Yes | Yes |
| List players | Yes | Yes | Optional later |

Permission checks must happen in Go application services, never in the LLM.

---

## 17. Outbox architecture

### 17.1 Why outbox

The app should not rely on immediate WhatsApp sends inside command handlers. Instead:

1. Domain service updates database.
2. Domain service inserts outgoing message.
3. Worker sends message.
4. Worker marks message as sent or failed.

### 17.2 Recommended settings

```text
OUTBOX_POLL_INTERVAL_MS=500
OUTBOX_BATCH_SIZE=20
OUTBOX_MAX_ATTEMPTS=5
OUTBOX_RETRY_BASE_SECONDS=5
OUTBOX_RETRY_MAX_SECONDS=300
OUTBOX_LOCK_TIMEOUT_SECONDS=30
```

### 17.3 Sender flow

```mermaid
flowchart TD
    Pending[Pending outgoing_messages] --> Lock[Lock batch]
    Lock --> Send[Send via whatsmeow]
    Send --> Success{Success?}
    Success -->|Yes| MarkSent[Mark sent]
    Success -->|No| Retry{Attempts left?}
    Retry -->|Yes| Reschedule[Set retry time]
    Retry -->|No| Failed[Mark failed]
```

---

## 18. Deployment architecture

### 18.1 Local development

```mermaid
flowchart TD
    Dev[Developer Machine] --> API[api container]
    Dev --> Admin[admin endpoint]
    Bot[bot container] --> WA[WhatsApp]
    Worker[worker container] --> Bot

    API --> AppDB[(postgres-app)]
    Bot --> AppDB
    Worker --> AppDB
    Bot --> WADB[(postgres-whatsmeow)]
```

### 18.2 Services

Recommended Docker Compose services:

```text
postgres-app
postgres-whatsmeow
api
bot
worker
admin optional
```

### 18.3 Recommended container settings

```text
restart: unless-stopped
healthcheck enabled for api, bot, worker, postgres
read-only filesystem where practical later
non-root user later
persistent volumes for both Postgres databases
```

### 18.4 Production first version

```mermaid
flowchart TD
    Internet --> Proxy[Caddy / NGINX]
    Proxy --> API[Connect API]
    Proxy --> Admin[Protected Admin Endpoints]

    Bot[WhatsApp Bot] --> WhatsApp[WhatsApp]
    API --> DB[(PostgreSQL App DB)]
    Worker --> DB
    Bot --> DB
    Bot --> WADB[(WhatsApp Store DB)]
```

The API should not directly call the bot for normal domain operations. It should call application services and write to the outbox.

---

## 19. Environment settings

Recommended `.env.example`:

```env
APP_ENV=development
APP_NAME=AirTrak
APP_DEFAULT_LOCALE=pt-BR
APP_SUPPORTED_LOCALES=pt-BR,en
APP_DEFAULT_TIMEZONE=America/Maceio
APP_PUBLIC_BASE_URL=http://localhost:8080

APP_DATABASE_URL=postgres://airtrak:airtrak@postgres-app:5432/airtrak?sslmode=disable
WA_DATABASE_URL=postgres://whatsmeow:whatsmeow@postgres-whatsmeow:5432/whatsmeow?sslmode=disable

API_ADDR=:8080
API_AUTH_MODE=admin_token
API_ADMIN_TOKEN=change-me
API_REQUEST_TIMEOUT_SECONDS=15
API_ENABLE_REFLECTION=true
API_ENABLE_HEALTH=true
API_CORS_ALLOWED_ORIGINS=http://localhost:5173,http://localhost:3000

WA_ENABLED=true
WA_PROCESS_ONLY_WHEN_MENTIONED=true
WA_IGNORE_NON_COMMAND_GROUP_MESSAGES=true
WA_BOT_MENTION_ALIASES=airtrak,bot
WA_SEND_RATE_LIMIT_PER_SECOND=1
WA_SEND_MAX_RETRIES=5
WA_SINGLE_INSTANCE_LOCK=true

LLM_ENABLED=true
LLM_PROVIDER=openai
LLM_MODEL=gpt-4o-mini
LLM_TEMPERATURE=0
LLM_MAX_OUTPUT_TOKENS=700
LLM_TIMEOUT_SECONDS=8
LLM_MAX_RETRIES=1
LLM_MIN_CONFIDENCE=0.65
LLM_LOG_RAW_MESSAGES=false
LLM_LOG_PARSED_PAYLOAD=true
LLM_PROMPT_VERSION=airtrak-intent-v1
OPENAI_API_KEY=

OUTBOX_POLL_INTERVAL_MS=500
OUTBOX_BATCH_SIZE=20
OUTBOX_MAX_ATTEMPTS=5
OUTBOX_RETRY_BASE_SECONDS=5
OUTBOX_RETRY_MAX_SECONDS=300

CONVERSATION_STATE_TTL_MINUTES=15
PENDING_ACTION_TTL_MINUTES=15

LOG_LEVEL=debug
LOG_FORMAT=json
METRICS_ENABLED=true
METRICS_ADDR=:9090
```

---

## 20. Observability

### 20.1 Logs

Use structured JSON logs.

Recommended fields:

```text
timestamp
level
service
trace_id
chat_jid
sender_jid_hash
message_id
intent
parser_source
confidence
group_id
game_id
error
```

Avoid logging raw WhatsApp text by default in production.

### 20.2 Metrics

Recommended metrics:

```text
whatsapp_messages_received_total
whatsapp_messages_ignored_total
commands_parsed_total
llm_parse_requests_total
llm_parse_errors_total
llm_parse_latency_ms
llm_input_tokens_total
llm_output_tokens_total
intent_resolution_errors_total
outgoing_messages_sent_total
outgoing_messages_failed_total
outbox_pending_count
games_created_total
registrations_created_total
```

### 20.3 Health checks

```text
POST /airtrak.health.v1.HealthService/Healthz   process is alive
POST /airtrak.health.v1.HealthService/Readyz    database reachable and service ready
GET /metrics                                      Prometheus metrics if enabled
```

---

## 21. Security and privacy

### 21.1 MVP security

- Protect API with an admin token initially.
- Protect admin endpoints separately.
- Never expose QR endpoint publicly without authentication.
- Do not log raw WhatsApp messages by default.
- Store `whatsmeow` session data in a persistent database with backups.
- Use HTTPS in production.
- Use rate limits on API and bot commands.

### 21.2 WhatsApp privacy

The bot should only process messages that are commands, mentions, replies to the bot, or part of active state. This minimizes accidental processing of private group conversations.

### 21.3 LLM privacy

Recommended default:

```text
LLM_LOG_RAW_MESSAGES=false
```

Send only the minimal context required:

- current time;
- timezone;
- group locale;
- known fields;
- upcoming games;
- sender registration status;
- user message being parsed.

Do not send full chat history unless you explicitly add that feature later.

---

## 22. Testing strategy

### 22.1 Unit tests

Test modules independently:

- fixed command parser;
- LLM schema validation;
- resolver;
- permissions;
- i18n rendering;
- date parsing;
- repository queries.

### 22.2 Integration tests

Use PostgreSQL through Docker/Testcontainers.

Test:

- setup group;
- create field;
- create game;
- join game;
- leave game;
- capacity and waitlist behavior;
- pending actions;
- conversation state;
- outbox retries.

### 22.3 Parser tests

Create golden test files:

```text
testdata/parser/pt-BR/create_game.json
testdata/parser/pt-BR/join_game.json
testdata/parser/en/create_game.json
testdata/parser/en/join_game.json
```

For LLM tests:

- use `MockProvider` for deterministic CI;
- optionally run real provider tests manually with `LLM_TEST_REAL_PROVIDER=true`.

### 22.4 API tests

Test every Connect service method.

Use API tests to validate the same operations exposed via WhatsApp.

### 22.5 Bot tests

Use fake incoming WhatsApp event objects and a fake sender.

Do not require real WhatsApp for CI.

---

## 23. CI recommendations

Recommended CI checks:

```text
go test ./...
go vet ./...
gofmt check
golangci-lint
buf lint
buf breaking against main
sqlc generate check
migration up/down test
```

For GitHub Actions or similar:

```text
jobs:
  backend-test
  proto-lint
  proto-breaking
  docker-build
```

---

## 24. Implementation plan for Codex and GPT-5.5

This section is designed to be used as an implementation plan with Codex and GPT-5.5 models.

### 24.1 How to use this plan with Codex

For each task:

1. Give Codex only one phase or one small task at a time.
2. Ask it to inspect the repository first.
3. Ask it to create or update tests before or alongside implementation.
4. Ask it to avoid large unrelated refactors.
5. Ask it to run the relevant tests.
6. Ask it to summarize changed files and any assumptions.

Recommended task prompt template:

```text
You are implementing AirTrak according to docs/architecture.md.

Task:
<one specific task>

Constraints:
- Keep changes minimal and focused.
- Add or update tests.
- Do not introduce unrelated dependencies.
- Do not implement future phases unless required.
- Preserve existing public interfaces unless this task explicitly changes them.

Acceptance criteria:
<clear checklist>

After implementation:
- Run relevant tests.
- Report changed files.
- Report any unresolved issues.
```

### 24.2 Definition of done for every phase

A phase is done only when:

- code compiles;
- tests pass;
- migrations are included if schema changed;
- `.env.example` is updated if config changed;
- docs are updated if behavior changed;
- no raw secrets are committed;
- no real WhatsApp or real LLM dependency is required for CI.

---

# Phase 0 — Repository foundation

## Goal

Create the initial repository structure and development standards.

## Tasks

1. Create root structure:
   - `backend/`
   - `proto/`
   - `docs/`
   - `.env.example`
   - `docker-compose.yml`
   - `Makefile`
   - `AGENTS.md`
2. Add `docs/architecture.md` using this document as source.
3. Create Go module under `backend/`.
4. Add basic config loader.
5. Add basic logger.
6. Add `make test`, `make lint`, `make proto`, `make migrate-up` placeholders.

## Acceptance criteria

- `go test ./...` runs successfully, even if there are no meaningful tests yet.
- `.env.example` exists with all MVP settings.
- `AGENTS.md` contains project rules for Codex.

## Suggested Codex prompt

```text
Set up the initial AirTrak repository foundation according to docs/architecture.md Phase 0. Create the Go module, folder layout, config loader, logger skeleton, .env.example, Makefile and AGENTS.md. Do not implement WhatsApp, API or DB logic yet. Add minimal tests for config loading.
```

---

# Phase 1 — Database foundation

## Goal

Create application database migrations and repository foundation.

## Tasks

1. Choose migration tool:
   - recommended: `goose` or `golang-migrate`.
2. Create migrations for:
   - `whatsapp_identities`
   - `groups`
   - `group_members`
   - `fields`
   - `games`
   - `game_slot_types`
   - `game_registrations`
   - `incoming_messages`
   - `outgoing_messages`
   - `pending_actions`
   - `conversation_states`
   - `llm_parse_logs`
   - `audit_events`
3. Add DB connection package using `pgxpool`.
4. Add repository interfaces and minimal implementations.
5. Add Docker Compose with `postgres-app` and `postgres-whatsmeow`.

## Acceptance criteria

- Migrations run against local Postgres.
- Tests can create a clean database and verify core tables exist.
- No `whatsmeow` tables are manually created in the app migrations.

## Suggested Codex prompt

```text
Implement Phase 1 database foundation. Add migrations for the MVP schema, pgxpool connection setup, repository skeletons, and docker-compose Postgres services. Include tests that run migrations against a test Postgres database. Keep business logic out of this phase.
```

---

# Phase 2 — Domain services and permissions

## Goal

Implement core business operations independent of WhatsApp, API and LLM.

## Tasks

1. Implement identity service:
   - create/get WhatsApp identity by JID.
2. Implement group service:
   - setup group;
   - get group;
   - set locale;
   - add/remove admin;
   - list members.
3. Implement field service:
   - create/list/update/delete fields.
4. Implement game service:
   - create/list/get/cancel games;
   - generate readable game codes.
5. Implement registration service:
   - join;
   - leave;
   - handle slot type;
   - handle full capacity and waitlist if included now.
6. Implement centralized permission service.
7. Implement audit event writes.
8. Implement outgoing message repository, but not WhatsApp sending yet.

## Acceptance criteria

- Domain services can be tested without WhatsApp.
- Permission tests cover owner/admin/member behavior.
- Game creation with rental and own AEG slots is tested.
- Registration uniqueness is tested.

## Suggested Codex prompt

```text
Implement Phase 2 domain services and permission layer. Keep this independent from WhatsApp, Connect RPC and LLM. Add tests for setup group, roles, field creation, game creation with slot types, join, leave, and permission denial cases.
```

---

# Phase 3 — i18n renderer

## Goal

Add translation rendering for `pt-BR` and `en`.

## Tasks

1. Add i18n package.
2. Add `pt-BR.toml` and `en.toml`.
3. Add locale resolution:
   - sender locale if present;
   - group locale;
   - default locale.
4. Add helper for rendering bot replies.
5. Add tests for required message keys.

## Acceptance criteria

- All required translation keys exist in both languages.
- Missing key test fails clearly.
- Dynamic template variables are tested.

## Suggested Codex prompt

```text
Implement Phase 3 i18n. Add pt-BR and en translation files, a localizer package, locale resolution, and tests that ensure all required keys exist in both locales and render with variables.
```

---

# Phase 4 — Connect RPC API

## Goal

Expose all domain operations through API before adding WhatsApp complexity.

## Tasks

1. Add Buf config.
2. Create initial Protobuf files:
   - auth optional placeholder;
   - groups;
   - fields;
   - games;
   - registrations;
   - parser placeholder;
   - bot placeholder.
3. Generate Go Connect server code.
4. Implement API handlers calling domain services.
5. Add admin-token auth interceptor.
6. Add health/readiness endpoints.
7. Add API integration tests.

## Acceptance criteria

- `buf lint` passes.
- Go code generation works.
- API can setup group, create field, create game and join game.
- API tests use generated clients where possible.
- Business logic remains in services, not handlers.

## Suggested Codex prompt

```text
Implement Phase 4 Connect RPC API. Add Buf/Protobuf definitions for groups, fields, games and registrations, generate Go code, implement handlers that call the existing domain services, add admin-token auth, health/readiness, and integration tests. Keep handlers thin.
```

---

# Phase 5 — Fixed command parser

## Goal

Add deterministic command parsing before LLM.

## Tasks

1. Define `ParsedIntent` types.
2. Implement fixed parser for:
   - `/setup`
   - `/config idioma pt-BR`
   - `/config language en`
   - `/jogos` / `/games`
   - `/jogo criar ...`
   - `/game create ...`
   - `/entrar` / `/join`
   - `/sair` / `/leave`
   - `/jogadores` / `/players`
   - `/jogo cancelar` / `/game cancel`
3. Add parser tests for `pt-BR` and `en`.
4. Implement resolver integration for fixed commands.

## Acceptance criteria

- Fixed commands produce `ParsedIntent` objects.
- Parser has golden tests.
- Resolver can execute fixed command intents through domain services.

## Suggested Codex prompt

```text
Implement Phase 5 fixed command parser. Define ParsedIntent and parse all MVP commands in pt-BR and en. Add golden tests. Integrate ParsedIntent with the resolver so fixed command intents can call the domain services.
```

---

# Phase 6 — Conversation state and pending actions

## Goal

Support confirmations and multi-step choices.

## Tasks

1. Implement conversation state repository/service.
2. Implement pending action repository/service.
3. Add confirmation parser for localized yes/no.
4. Add resolver flows for:
   - confirm create game;
   - confirm cancel game;
   - select game;
   - select slot type.
5. Add expiration cleanup logic.

## Acceptance criteria

- Natural-language-like pending action can be confirmed with `sim` or `yes`.
- Expired pending actions cannot execute.
- Multiple matching games produce a selection state.
- Selection by number works.

## Suggested Codex prompt

```text
Implement Phase 6 conversation state and pending actions. Add localized confirmation handling, state expiration, pending action execution, and tests for confirm create game, cancel game, select game and select slot type.
```

---

# Phase 7 — LLM provider and natural-language parser

## Goal

Add generic LLM parsing without tying business logic to OpenAI.

## Tasks

1. Create `LLMProvider` interface.
2. Create `MockProvider` for tests.
3. Create `OpenAIProvider` behind config.
4. Define JSON schema for parser output.
5. Add prompt builder.
6. Add LLM parser using provider.
7. Validate output schema.
8. Add parse logs.
9. Add parser service API methods:
   - `ParseInput`
   - `PreviewResolvedIntent`.

## Acceptance criteria

- CI can run with `MockProvider` only.
- Real provider is optional and config-gated.
- Invalid LLM JSON is handled safely.
- Low confidence returns help/follow-up, not action.
- ParserService can test natural language without WhatsApp.

## Suggested Codex prompt

```text
Implement Phase 7 generic LLM parsing. Add LLMProvider, MockProvider, OpenAIProvider skeleton/config, strict parser output schema, prompt builder, LLM parser, parse logs, and ParserService methods. Tests must use MockProvider and cover success, low confidence and invalid JSON.
```

---

# Phase 8 — WhatsApp integration with `whatsmeow`

## Goal

Connect the bot to WhatsApp and route messages through the existing input pipeline.

## Tasks

1. Add `whatsmeow` client setup.
2. Configure SQL store using `WA_DATABASE_URL`.
3. Implement QR/pairing status.
4. Implement event handler for text messages.
5. Implement filtering rules:
   - slash command;
   - mention;
   - reply to bot;
   - active state.
6. Persist incoming message IDs for idempotency.
7. Call input router.
8. Queue replies in outbox.
9. Add fake event tests.

## Acceptance criteria

- Bot can connect and expose QR.
- Bot ignores normal group messages.
- Bot processes `/help` and `/setup`.
- Duplicate message ID is ignored.
- Tests do not require real WhatsApp.

## Suggested Codex prompt

```text
Implement Phase 8 whatsmeow integration. Add client setup with sqlstore, QR/status handling, message event handler, filtering rules, incoming message idempotency and input router integration. Use fake event tests; do not require real WhatsApp in CI.
```

---

# Phase 9 — Outbox worker and WhatsApp sender

## Goal

Send queued replies reliably.

## Tasks

1. Implement outbox polling worker.
2. Add row locking for batches.
3. Implement retry policy.
4. Implement WhatsApp sender adapter.
5. Add send rate limiting.
6. Add metrics for sent/failed/pending messages.

## Acceptance criteria

- Pending messages are sent and marked `sent`.
- Failed messages retry and eventually become `failed`.
- Worker handles no pending messages efficiently.
- Tests use fake sender.

## Suggested Codex prompt

```text
Implement Phase 9 outbox worker. Add batch polling, locking, retry scheduling, fake sender tests, whatsmeow sender adapter, rate limiting, and metrics hooks. Keep WhatsApp-specific sending behind an interface.
```

---

# Phase 10 — Bot/admin operational API

## Goal

Expose operational endpoints for development and production management.

## Tasks

1. Add `BotService` Connect API methods:
   - status;
   - QR;
   - reconnect.
2. Add HTTP health/readiness if not already done.
3. Add metrics endpoint.
4. Protect admin/bot endpoints with admin token.

## Acceptance criteria

- QR/status are accessible only with auth.
- Health endpoints do not leak sensitive data.
- API reports database readiness.

## Suggested Codex prompt

```text
Implement Phase 10 bot/admin operational API. Add BotService for WhatsApp status, QR and reconnect, protect it with admin-token auth, and add health/readiness/metrics endpoints if missing.
```

---

# Phase 11 — End-to-end MVP flows

## Goal

Validate the complete MVP behavior.

## Tasks

1. End-to-end test using API:
   - setup group;
   - create field;
   - create game;
   - join;
   - leave;
   - cancel.
2. End-to-end test using fixed parser:
   - `/setup`;
   - `/jogo criar`;
   - `/entrar`;
   - `/sair`.
3. End-to-end test using LLM mock parser:
   - create game natural language;
   - confirm;
   - join natural language;
   - ambiguous join selection.
4. Manual real WhatsApp test checklist.

## Acceptance criteria

- All core flows pass without real WhatsApp or real LLM in CI.
- Manual checklist documents how to test with real WhatsApp.
- All output messages are localized.

## Suggested Codex prompt

```text
Implement Phase 11 end-to-end MVP tests. Cover API flows, fixed parser flows and LLM mock natural-language flows through the resolver and domain services. Add a manual WhatsApp testing checklist in docs/manual-test.md.
```

---

# Phase 12 — Hardening and release preparation

## Goal

Prepare for a first open-source release.

## Tasks

1. Add README with quickstart.
2. Add Docker Compose quickstart.
3. Add license.
4. Add contribution guide.
5. Add command reference.
6. Add API reference generation notes.
7. Add backup notes for Postgres and WhatsApp store.
8. Add security/privacy notes.
9. Add CI pipeline.
10. Add release checklist.

## Acceptance criteria

- New developer can run the project locally from README.
- `.env.example` is complete.
- Docs explain WhatsApp limitations and unofficial nature of `whatsmeow`.
- CI passes.

## Suggested Codex prompt

```text
Implement Phase 12 release preparation. Add README quickstart, command reference, contribution guide, security/privacy notes, backup notes, CI workflow and release checklist. Do not add new features.
```

---

## 25. Recommended `AGENTS.md` for Codex

Create an `AGENTS.md` file with rules like:

```text
# AirTrak Agent Instructions

## Project architecture
- Go modular monolith.
- Domain services are the source of truth.
- WhatsApp, Connect API and future clients are adapters.
- Do not put business rules in handlers or WhatsApp event callbacks.
- LLM output is never trusted directly; always validate and resolve in Go.

## Development rules
- Keep changes focused on the requested task.
- Add or update tests for every behavior change.
- Do not introduce new dependencies without explaining why.
- Do not commit secrets.
- Do not require real WhatsApp or real LLM access in CI.
- Use MockProvider/FakeSender in tests.

## API rules
- Connect RPC is the primary API.
- Keep Protobuf packages versioned under airtrak.<domain>.v1.
- Do not reuse field numbers after removal; reserve them.
- Run buf lint and breaking checks when proto changes.

## Database rules
- Every schema change needs a migration.
- Keep whatsmeow store separate from app migrations.
- Use transactions for multi-table domain operations.
- Use idempotency for incoming WhatsApp messages.

## i18n rules
- User-facing messages must use translation keys.
- pt-BR and en must have the same key set.

## Privacy rules
- Do not log raw WhatsApp messages by default.
- Do not send unnecessary context to the LLM.
```

---

## 26. First vertical slice to implement

The best first vertical slice is:

```text
/setup via fixed command
  -> creates WhatsApp identity
  -> creates group
  -> creates owner membership
  -> sends localized confirmation through outbox
```

Why this slice:

- touches WhatsApp identity;
- touches group setup;
- touches permissions;
- touches i18n;
- touches outbox;
- avoids LLM complexity;
- proves the project shape.

Second vertical slice:

```text
/jogo criar fixed command
  -> create field if known or require field
  -> create game
  -> create slot types
  -> send localized reply
```

Third vertical slice:

```text
@bot crie jogo... natural language
  -> LLM parser
  -> resolver
  -> pending confirmation
  -> user says sim
  -> create game
```

---

## 27. References

- Connect RPC documentation: https://connectrpc.com/docs/introduction/
- Connect Go getting started: https://connectrpc.com/docs/go/getting-started/
- Connect Web getting started: https://connectrpc.com/docs/web/getting-started/
- Connect protocol reference: https://connectrpc.com/docs/protocol/
- Buf breaking change detection: https://buf.build/docs/breaking/
- OpenAI Structured Outputs: https://platform.openai.com/docs/guides/structured-outputs
- OpenAI Prompt Caching: https://platform.openai.com/docs/guides/prompt-caching
- whatsmeow GitHub: https://github.com/tulir/whatsmeow
- whatsmeow sqlstore package: https://pkg.go.dev/go.mau.fi/whatsmeow/store/sqlstore
