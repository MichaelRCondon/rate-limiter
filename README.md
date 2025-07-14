# Simple Go Rate Limiter

This is built mostly as an excuse to learn Go, so it's confined to a relatively small footprint. The system is designed to be distributed, scalable, and deployable as a demo in a small set of docker containers.

**This is very much a work-in-progress** - not all features are implemented yet, but it's functional enough to demonstrate rate limiting concepts.

## What It Does

I've built a reverse proxy that sits in front of your backend API and enforces rate limits. It uses Redis to track request counts across distributed instances, so you can scale horizontally without losing rate limiting state.

The cool part is that you can swap rate limiting algorithms just by changing the config - no code changes needed. Right now I have two algorithms implemented:

- **Permissive** (`allow_all`) - Basically turns off rate limiting, useful for development
- **Bucketed Sliding Window** (`bucketed_sliding_window`) - A memory-efficient sliding window that uses 1-minute buckets

The system supports JWT-based authentication so you can have different rate limits per account, though I'm still working on making that fully configurable.

## Getting Started

The easiest way to try this out is with Docker Compose. I've set up everything you need including a demo API and Redis:

```bash
# Linux/macOS
./start-local.sh

# Windows  
start-local.bat
```

This will generate secure credentials and start:
- Rate limiter proxy on http://localhost:8080
- Demo API on http://localhost:9080/hello  
- Redis Commander UI on http://localhost:8081

## Configuration

The rate limiter reads its config from a JSON file. Here's what mine looks like:

```json
{
  "default_limit_count": 100,
  "default_period": "1h", 
  "algorithm": "bucketed_sliding_window",
  "redis_config": {
    "redis_url": "localhost:6379",
    "db": 0
  },
  "server_config": {
    "port": 8080,
    "read_timeout": "30s",
    "write_timeout": "30s", 
    "idle_timeout": "120s"
  },
  "backend_config": {
    "backend_url": "http://localhost:9080",
    "backend_healthcheck_url": "http://localhost:9080/health"
  }
}
```

You can switch algorithms by changing the `algorithm` field. I plan to add more algorithms as I learn about different approaches.

## JWT Token Generation

I built a little tool to generate JWT tokens for testing. It's in the `tools/jwt-signer` directory:

```bash
cd tools/jwt-signer
go build -o jwt-signer

# Generate a token for a test user
./jwt-signer -secret="your-jwt-secret" -preset=user1

# Get it formatted as a curl command
./jwt-signer -secret="your-jwt-secret" -preset=user1 -output=curl

# See what presets are available
./jwt-signer -list
```

The presets I've set up are:
- `user1`: Regular user (AccountID: 12345)
- `user2`: Another regular user (AccountID: 67890)  
- `admin1`: Admin user (AccountID: 99999)

## Testing It Out

Try making some requests to see the rate limiting in action:

```bash
# Basic request (uses default account)
curl http://localhost:8080/hello

# With a JWT token
curl -H "Authorization: Bearer YOUR_JWT_TOKEN" http://localhost:8080/hello

# Spam it to trigger rate limiting
for i in {1..10}; do curl http://localhost:8080/hello; done
```

## How It Works

The flow is pretty straightforward:
1. Request comes in to the rate limiter
2. I extract the account ID from the JWT (or use a default)
3. Check Redis to see if this account has exceeded their limit
4. If they're under the limit, forward the request to the backend
5. If they're over the limit, return a rate limit error

## What I'm Planning to Build Next

I've got a bunch of TODOs scattered throughout the code for features I want to add:

### Authentication & Account Management
- Finish implementing the JWT authentication system properly
- Load per-account rate limits from Redis instead of using the default for everyone
- Add proper role-based access control
- Build a simple web UI to manage account settings and reset limits

### More Rate Limiting Algorithms  
- Implement a true continuous sliding window (no bucketing)
- Add configuration options for bucket count and Redis key prefixes
- Support for different rate limit windows per account

### Backend & Deployment
- Support multiple backend services with intelligent routing
- Create a Helm chart so this can be easily deployed to Kubernetes
- Better health checking and graceful shutdown
- Improve configuration validation (right now it's pretty basic)

### Operations & Monitoring
- Clean up URL logging to remove credentials  
- Better error handling throughout the system
- More comprehensive logging and metrics

## Manual Setup

If you don't want to use Docker Compose, you can run everything manually:

```bash
# Start Redis
docker run -d -p 6379:6379 redis:7-alpine

# Copy and edit the config
cp application_config.json my_config.json
# Edit my_config.json as needed

# Run the rate limiter
go run main.go -config=my_config.json
```

If this grows beyond a learning project and becomes actually useful, I'll add proper documentation and maybe even tests. For now, it's functional enough to demonstrate the concepts and let me experiment with different rate limiting approaches in Go.

