# Communication Channels Setup Guide

This guide provides complete setup instructions for the WhatsApp and USSD communication channels.

## Overview

The Zimbabwe Loyalty Platform now includes two communication channels:
1. **WhatsApp Business API** - For rich customer interactions
2. **USSD** - For basic feature phone access

## Architecture

### WhatsApp Integration

```
WhatsApp Cloud API
       ↓
GET/POST /public/wa/webhook
       ↓
Signature Verification
       ↓
Message Processor
   ↓     ↓     ↓
Commands  Flows  Session Management
       ↓
Database (wa_sessions, customers, issuances)
       ↓
Message Sender (HTTP Client)
       ↓
WhatsApp Cloud API
```

### USSD Integration

```
Mobile Network Operator
       ↓
POST /public/ussd/callback
       ↓
USSD Handler
       ↓
Menu System
   ↓     ↓
Session   Menu
Manager   Navigation
       ↓
Database (ussd_sessions, customers, issuances)
       ↓
Response Builder
       ↓
Mobile Network Operator
```

## Files Implemented

### WhatsApp Channel
- `/api/internal/channels/whatsapp/types.go` - Data structures
- `/api/internal/channels/whatsapp/webhook.go` - Webhook handler
- `/api/internal/channels/whatsapp/processor.go` - Message processor
- `/api/internal/channels/whatsapp/sender.go` - Message sender
- `/api/internal/channels/whatsapp/session.go` - Session management
- `/api/internal/channels/whatsapp/templates.go` - Message templates
- `/api/internal/channels/whatsapp/webhook_test.go` - Tests

### USSD Channel
- `/api/internal/channels/ussd/types.go` - Data structures
- `/api/internal/channels/ussd/handler.go` - USSD handler
- `/api/internal/channels/ussd/menus.go` - Menu system
- `/api/internal/channels/ussd/session.go` - Session management
- `/api/internal/channels/ussd/response.go` - Response builder
- `/api/internal/channels/ussd/handler_test.go` - Tests

### Integration Points
- `/api/internal/http/router.go` - Updated with channel handlers
- `/api/internal/config/config.go` - Added WhatsApp configuration
- `/.env.example` - Added WhatsApp environment variables

## WhatsApp Business API Setup

### Prerequisites
1. Facebook Business Account
2. WhatsApp Business Account
3. WhatsApp Business Phone Number
4. Meta Developer Account

### Step 1: Create WhatsApp Business App

1. Go to https://developers.facebook.com/
2. Create a new app or use existing
3. Add "WhatsApp" product to your app
4. Follow the setup wizard

### Step 2: Get Credentials

From the WhatsApp > API Setup page, collect:

1. **Phone Number ID**:
   - Found under "Phone number ID"
   - Example: `123456789012345`

2. **WhatsApp Business Account ID**:
   - Found in the URL or API Setup
   - Example: `234567890123456`

3. **Access Token**:
   - Generate a temporary token for testing
   - For production, generate a System User token with `whatsapp_business_messaging` permission
   - Example: `EAAx...` (long string)

4. **App Secret**:
   - Found in App Settings > Basic > App Secret
   - Click "Show" to reveal
   - Example: `abc123def456...`

5. **Verify Token**:
   - Create your own secure random string
   - This will be used for webhook verification
   - Example: `my_secure_verify_token_2024`

### Step 3: Configure Webhook

1. In WhatsApp > Configuration > Webhook
2. Set Callback URL: `https://your-domain.com/public/wa/webhook`
3. Set Verify Token: (use the same one you'll set in env)
4. Subscribe to webhook fields:
   - `messages`
   - `message_status` (optional)

### Step 4: Set Environment Variables

Add to your `.env` file:

```bash
# WhatsApp Business API Configuration
WHATSAPP_VERIFY_TOKEN=my_secure_verify_token_2024
WHATSAPP_APP_SECRET=abc123def456...
WHATSAPP_PHONE_NUMBER_ID=123456789012345
WHATSAPP_ACCESS_TOKEN=EAAx...
```

### Step 5: Create Message Templates

WhatsApp requires pre-approved templates for business-initiated messages. Create these templates in the WhatsApp Manager:

#### Template 1: loyalty_welcome
```
Welcome to our loyalty program, {{1}}! You can now earn rewards on every purchase.
```
Variables: Customer name

#### Template 2: reward_issued
```
Congratulations! You've earned {{1}}. Code: {{2}}. Valid until {{3}}.
```
Variables: Reward name, Code, Expiry date

#### Template 3: reward_reminder
```
Reminder: Your {{1}} (Code: {{2}}) expires in {{3}} days. Use it soon!
```
Variables: Reward name, Code, Days until expiry

#### Template 4: reward_redeemed
```
Your {{1}} has been successfully redeemed at {{2}}. Thank you!
```
Variables: Reward name, Location/date

### Step 6: Test Webhook

1. Start your API server
2. Use WhatsApp's "Test" button in the API Setup page
3. Check logs for incoming webhook
4. Send a test message to your WhatsApp Business number
5. Verify the bot responds

## WhatsApp Commands

Once set up, customers can use these commands:

- `/start` or `/enroll` - Join the loyalty program
- `/balance` - Check points balance (coming soon)
- `/rewards` - View available rewards
- `/myrewards` - See active rewards with codes
- `/redeem [code]` - Redeem a reward (e.g., `/redeem ABC123`)
- `/refer` - Get referral link (coming soon)
- `/help` - Show help message

## USSD Setup

### Prerequisites
1. Mobile Network Operator (MNO) partnership
2. USSD short code allocation
3. USSD gateway/aggregator (e.g., Africa's Talking)

### Using Africa's Talking

1. **Create Account**:
   - Go to https://africastalking.com/
   - Sign up and verify account
   - Get sandbox credentials

2. **Set Up USSD Code**:
   - Navigate to USSD section
   - Create a new USSD channel
   - Set callback URL: `https://your-domain.com/public/ussd/callback`
   - Get assigned short code (e.g., `*384*1234#`)

3. **Configure Callback**:
   - POST endpoint: `/public/ussd/callback`
   - Format: Form-encoded or JSON
   - Fields:
     - `sessionId` - Unique session identifier
     - `serviceCode` - USSD code dialed
     - `phoneNumber` - Customer's phone number
     - `text` - User input sequence
     - `networkCode` - Mobile network code (optional)

4. **Test USSD**:
   - Use Africa's Talking simulator
   - Or dial the USSD code from a test number
   - Navigate through the menus

### USSD Menu Structure

```
Main Menu
├── 1. My Rewards → Show active rewards
├── 2. Check Balance → Show points balance
├── 3. Redeem Reward → Enter code to redeem
└── 4. Help → Show help information
```

### USSD Response Format

The system responds with:
- `CON` - Continue session (shows menu and waits for input)
- `END` - End session (final message)

Example:
```
CON Welcome to Loyalty
1. My Rewards
2. Check Balance
3. Redeem Reward
4. Help
```

## Database Schema

The channels use these tables (already created in migrations):

### wa_sessions
```sql
CREATE TABLE wa_sessions (
  id            uuid PRIMARY KEY,
  tenant_id     uuid NOT NULL REFERENCES tenants(id),
  customer_id   uuid REFERENCES customers(id),
  wa_id         text NOT NULL,        -- WhatsApp ID (phone number)
  phone_e164    text NOT NULL,        -- E.164 format phone
  state         jsonb NOT NULL,       -- Session state (flow, step, data)
  last_msg_at   timestamptz NOT NULL,
  created_at    timestamptz NOT NULL
);
```

### ussd_sessions
```sql
CREATE TABLE ussd_sessions (
  id            uuid PRIMARY KEY,
  tenant_id     uuid NOT NULL REFERENCES tenants(id),
  customer_id   uuid REFERENCES customers(id),
  session_id    text NOT NULL,        -- From USSD provider
  phone_e164    text NOT NULL,        -- E.164 format phone
  state         jsonb NOT NULL,       -- Session state (menu, stack, data)
  last_input_at timestamptz NOT NULL,
  created_at    timestamptz NOT NULL
);
```

## Testing

### WhatsApp Tests

Run WhatsApp tests:
```bash
cd api
go test ./internal/channels/whatsapp/... -v
```

Key test cases:
- ✓ Webhook verification
- ✓ Signature verification
- ✓ Payload parsing
- ✓ Message text extraction
- ✓ Command parsing

### USSD Tests

Run USSD tests:
```bash
cd api
go test ./internal/channels/ussd/... -v
```

Key test cases:
- ✓ Request parsing
- ✓ Response formatting
- ✓ Menu option parsing
- ✓ Session data management
- ✓ Response builder
- ✓ Phone number normalization

### Manual Testing

#### WhatsApp
1. Send a message to your WhatsApp Business number
2. Try commands: `/enroll`, `/help`, `/myrewards`
3. Check logs for processing
4. Verify responses are received

#### USSD
1. Dial the USSD code from a phone
2. Navigate through menus (select options 1-4)
3. Test back navigation (option 0)
4. Verify session persistence

## Deployment

### Environment Variables

Production `.env`:
```bash
# WhatsApp Business API
WHATSAPP_VERIFY_TOKEN=<secure-random-token>
WHATSAPP_APP_SECRET=<from-meta-developer>
WHATSAPP_PHONE_NUMBER_ID=<from-whatsapp-setup>
WHATSAPP_ACCESS_TOKEN=<system-user-token>
```

### Webhook Security

1. **HTTPS Required**: WhatsApp requires HTTPS for webhooks
2. **Signature Verification**: Always verify HMAC signatures
3. **Rate Limiting**: Implement rate limiting to prevent abuse
4. **Logging**: Log all incoming messages for debugging

### Monitoring

Monitor these metrics:
- Webhook response time (should be < 3 seconds)
- Message delivery rate
- Session creation rate
- Error rates by type
- Command usage frequency

### Scaling

For high volume:
1. Use message queue for async processing (Redis/RabbitMQ)
2. Separate webhook receiver from processor
3. Scale horizontally with load balancer
4. Use database connection pooling
5. Cache frequently accessed data

## Troubleshooting

### WhatsApp Issues

**Webhook not receiving messages:**
- Check webhook URL is accessible publicly
- Verify HTTPS certificate is valid
- Check firewall allows Meta IP ranges
- Verify webhook subscriptions are active

**Signature verification fails:**
- Ensure `WHATSAPP_APP_SECRET` is correct
- Check for trailing whitespace in env vars
- Verify payload is being read correctly

**Messages not sending:**
- Check `WHATSAPP_ACCESS_TOKEN` is valid
- Verify phone number has `whatsapp_business_messaging` permission
- Check recipient has WhatsApp and hasn't blocked you
- Review Meta API error codes

### USSD Issues

**Session not persisting:**
- Check database connection
- Verify `session_id` is being sent correctly
- Check session cleanup isn't too aggressive

**Menu not displaying:**
- Verify response format is `CON` or `END`
- Check character limits (182 chars per message)
- Test response encoding (UTF-8)

**Phone number linking fails:**
- Ensure E.164 normalization is correct
- Check customer exists in database
- Verify tenant context is set

## Limitations

### WhatsApp
- Templates must be pre-approved (24-48 hour review)
- Rate limits apply (varies by business tier)
- 24-hour customer care window for free-form messages
- Media messages have size limits

### USSD
- Character limit: ~182 characters per screen
- No rich media (text only)
- Session timeout: typically 30-60 seconds of inactivity
- No message history (ephemeral)

## Next Steps

### Phase 4 Enhancements
- [ ] Multi-language support (English, Shona)
- [ ] Rich media messages (images, documents)
- [ ] Interactive buttons and lists
- [ ] Payment integration via WhatsApp
- [ ] Analytics dashboard
- [ ] A/B testing for messages
- [ ] Chatbot with NLP (DialogFlow/Rasa)

### Optimizations
- [ ] Message queue for async processing
- [ ] Redis cache for session data
- [ ] Webhook retry logic with exponential backoff
- [ ] Customer sentiment analysis
- [ ] Automated template management

## Support

For issues or questions:
1. Check logs: `docker-compose logs api`
2. Review Meta error codes: https://developers.facebook.com/docs/whatsapp/cloud-api/support/error-codes
3. Test with WhatsApp Business API simulator
4. Contact support@your-company.com

## References

- [WhatsApp Cloud API Docs](https://developers.facebook.com/docs/whatsapp/cloud-api)
- [Africa's Talking USSD Docs](https://developers.africastalking.com/docs/ussd)
- [E.164 Phone Number Format](https://en.wikipedia.org/wiki/E.164)
- [HMAC Signature Verification](https://developers.facebook.com/docs/graph-api/webhooks/getting-started#verification-requests)
