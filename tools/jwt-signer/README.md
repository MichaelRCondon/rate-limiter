# JWT Signer Tool

A command-line tool for generating JWT tokens for testing the rate limiter.

## Usage

### Basic Usage
```bash
# Build the tool
go build -o jwt-signer

# Generate token with preset user
./jwt-signer -secret="your-secret" -preset=user1

# Generate token with custom claims
./jwt-signer -secret="your-secret" -user="testuser" -account=12345 -role="user"
```

### Command Line Options

- `-secret`: JWT secret key (required)
- `-preset`: Use predefined user (user1, admin1, user2)
- `-user`: User ID/subject
- `-account`: Account ID for rate limiting
- `-role`: User role (user, admin)
- `-duration`: Token validity duration (default: 24h)
- `-output`: Output format (token, header, curl, json)
- `-list`: List available presets

### Presets

- `user1`: Standard user (AccountID: 12345, Role: user)
- `user2`: Standard user (AccountID: 67890, Role: user)  
- `admin1`: Admin user (AccountID: 99999, Role: admin)

### Output Formats

- `token`: Just the JWT token string
- `header`: Authorization header format
- `curl`: Complete curl command
- `json`: JSON with token and claims

### Examples

```bash
# List available presets
./jwt-signer -list

# Generate token for user1
./jwt-signer -secret="my-secret" -preset=user1

# Generate admin token with custom duration
./jwt-signer -secret="my-secret" -preset=admin1 -duration=1h

# Generate token as curl command
./jwt-signer -secret="my-secret" -preset=user1 -output=curl

# Generate custom token
./jwt-signer -secret="my-secret" -user="custom123" -account=55555 -role="user"
```