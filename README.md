<div align="center">

# 🔥 FireY

### Firebase Authorization Security Tester

*Detect misconfigurations in your Firebase security rules before attackers do*

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=for-the-badge&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-green.svg?style=for-the-badge)](LICENSE)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg?style=for-the-badge)](CONTRIBUTING.md)

[Features](#-features) • [Installation](#-installation) • [Usage](#-usage) • [Examples](#-examples) • [Documentation](#-documentation)

</div>

---

## 🎯 Why FireY?

Firebase security rules can be tricky. A simple misconfiguration can expose sensitive data. FireY helps you:

- ✅ **Test multiple endpoints** with different HTTP methods
- ✅ **Monitor changes** over time with Keep An Eye mode
- ✅ **Detect time-based rules** that open access during specific hours
- ✅ **Identify data leaks** by showing response sizes for successful requests
- ✅ **Run parallel tests** for faster security audits

---

## ✨ Features

<table>
<tr>
<td width="50%">

### 🔍 Normal Mode
Test specific Firebase paths with all HTTP methods (GET, POST, PATCH, DELETE) and instantly see which endpoints are accessible.

</td>
<td width="50%">

### 👁️ Keep An Eye Mode
Run continuous 24-hour monitoring with checks every 30 minutes. Perfect for detecting time-based security rule changes.

</td>
</tr>
<tr>
<td width="50%">

### ⚡ Multi-threaded Testing
Speed up your security audits with parallel request processing using the `-t` flag.

</td>
<td width="50%">

### 🎨 Color-Coded Results
Instantly identify issues with color-coded status codes:
- 🟢 **Green**: Success (potential leak!)
- 🟡 **Yellow**: Forbidden (protected)
- 🔴 **Red**: Error
- 🔵 **Cyan**: Not Found

</td>
</tr>
</table>

---

## 🚀 Installation

### Prerequisites
- Go 1.21 or higher

### Build from Source

```bash
# Clone the repository
git clone https://github.com/yourusername/firey.git
cd firey

# Build the binary
go build -o firey main.go

# Make it executable (Linux/macOS)
chmod +x firey

# Optional: Move to PATH
sudo mv firey /usr/local/bin/
```

### Quick Start

```bash
# Test a single path
./firey -i your-project-id -p /users/1

# Test multiple paths from a file
./firey -i your-project-id -l paths.txt

# Run 24-hour monitoring
./firey -i your-project-id -l paths.txt -kae -o security-monitor.txt
```

---

## 📖 Usage

### Command Line Flags

| Flag | Description | Required | Example |
|------|-------------|----------|---------|
| `-i` | Firebase Project ID | ✅ Yes | `-i my-firebase-project` |
| `-p` | Single path to test | No* | `-p /users/123` |
| `-l` | File containing paths (one per line) | No* | `-l paths.txt` |
| `-m` | HTTP methods to test | No | `-m GET,POST` |
| `-u` | Custom base URL | No | `-u https://custom.url` |
| `-v` | Verbose output with details | No | `-v` |
| `-o` | Output file for results | No | `-o results.txt` |
| `-kae` | Keep An Eye mode (24h monitoring) | No | `-kae` |
| `-t` | Number of parallel threads | No | `-t 10` |
| `-s` | Silent mode (no banner) | No | `-s` |

*Either `-p` or `-l` must be provided*

### Creating a Paths File

Create a `paths.txt` file with one path per line:

```txt
# Admin endpoints
/admin
/admin/users
/admin/config

# User data
/users/1
/users/profile

# Sensitive data
/secrets
/internal/config
```

Lines starting with `#` are treated as comments and ignored.

---

## 🎬 Examples

### Basic Security Audit

```bash
./firey -i production-db -p /admin
```

**Output:**
```
[GET] /admin -> 200 SUCCESS (length: 1234 bytes)  🚨 POTENTIAL LEAK!
[POST] /admin -> 403 FORBIDDEN ✓
[DELETE] /admin -> 403 FORBIDDEN ✓
[PATCH] /admin -> 403 FORBIDDEN ✓
```

### Test Multiple Endpoints

```bash
./firey -i my-project -l paths.txt -v -o audit.txt
```

### Monitor for Time-Based Rules

Some developers open access during specific hours for migrations or backups. Catch these windows:

```bash
./firey -i prod-db -l critical-paths.txt -kae -o monitoring.txt
```

This runs for 24 hours, checking every 30 minutes. Check status:

```bash
cat .firey_status.json
```

```json
{
  "pid": 12345,
  "start_time": "2024-11-30T10:00:00Z",
  "next_check": "2024-11-30T10:30:00Z",
  "iteration": 3
}
```

### Fast Parallel Scan

```bash
./firey -i my-project -l large-paths.txt -t 20 -s -o results.txt
```

Test hundreds of endpoints quickly with 20 parallel threads.

### Test Specific Methods Only

```bash
./firey -i my-project -l paths.txt -m GET,DELETE -v
```

Only test read and delete permissions.

### Custom Database

```bash
./firey -i my-project \
  -u https://firestore.googleapis.com/v1/projects/my-project/databases/custom/documents \
  -p /test
```

---

## 📊 Output Format

### Normal Mode

```
[GET] /users -> 403 FORBIDDEN
[POST] /admin -> 401 UNAUTHORIZED
[DELETE] /public -> 200 SUCCESS (length: 1234 bytes)
[GET] /data -> 200 SUCCESS (length: 567 bytes)

========================================
Summary
========================================
200 SUCCESS: 7
403 FORBIDDEN: 12
404 NOT_FOUND: 3

Total: 22
```

### Verbose Mode

Detailed information including timestamps, full responses, and body content:

```json
{
  "timestamp": "2024-11-30T10:30:00Z",
  "url": "https://firestore.googleapis.com/v1/projects/my-project/databases/(default)/documents/users/1",
  "path": "/users/1",
  "method": "GET",
  "status_code": 200,
  "status": "SUCCESS",
  "body_length": 1234,
  "body": "{\"user\": \"data\"}"
}
```

### Color Coding

- 🟢 **Green (200 SUCCESS)**: Shows response length - indicates data exposure!
- 🟡 **Yellow (403 FORBIDDEN, 401 UNAUTHORIZED)**: Properly protected endpoints
- 🔵 **Cyan (404 NOT_FOUND)**: Path doesn't exist
- 🔴 **Red (4xx/5xx ERRORS)**: Something went wrong

---

## 🛡️ Security Use Cases

### 1. Pre-Deployment Testing

```bash
# Test all endpoints before deploying new rules
./firey -i staging-project -l all-endpoints.txt -v -o pre-deploy-audit.txt
```

### 2. Production Monitoring

```bash
# Continuous monitoring of critical paths
./firey -i prod-db -l critical.txt -kae -o prod-monitor.txt &
```

### 3. Penetration Testing

```bash
# Fast scan with verbose output
./firey -i target-project -l discovered-paths.txt -t 50 -v -o pentest-results.txt
```

### 4. Detecting Time-Based Misconfigurations

```bash
# 24-hour monitoring for time-based rules
./firey -i prod-db -l admin-paths.txt -kae -o time-based-check.txt
```

Check if any endpoints become accessible during off-hours, maintenance windows, or backup periods.

---

## 💡 Pro Tips

1. **Start Small**: Test a few paths first to verify connectivity
   ```bash
   ./firey -i my-project -p /test
   ```

2. **Use Verbose Mode**: For detailed analysis of responses
   ```bash
   ./firey -i my-project -l paths.txt -v | grep SUCCESS
   ```

3. **Monitor Background Jobs**: Keep an eye on long-running monitors
   ```bash
   watch -n 60 'cat .firey_status.json'
   ```

4. **Analyze Results**: Use grep to find specific issues
   ```bash
   grep "200 SUCCESS" results.txt  # Find all accessible endpoints
   grep "length:" results.txt      # Check data leak sizes
   ```

5. **Combine with Other Tools**: Pipe results for further analysis
   ```bash
   ./firey -i my-project -l paths.txt | tee results.txt | grep -E "(200|403)"
   ```

---

## 🐛 Troubleshooting

### Common Issues

**"Error reading paths file"**
- Ensure the file exists and is readable
- Check file path is correct

**"No paths provided"**
- Use either `-p` for a single path or `-l` for a file

**Network timeout errors**
- Check your internet connection
- Verify the project ID is correct
- Try reducing thread count with `-t 1`

**Permission denied errors**
- Make sure the binary is executable: `chmod +x firey`

---

## 📚 Documentation

### How Firebase Security Testing Works

FireY sends HTTP requests to your Firebase Firestore REST API endpoints and analyzes the responses:

1. **200 OK**: Endpoint is accessible (potential security issue!)
2. **403 Forbidden**: Security rules are blocking access (good!)
3. **401 Unauthorized**: Authentication required
4. **404 Not Found**: Path doesn't exist

### Understanding Results

- **Large response size**: Indicates data is being returned (data leak!)
- **Multiple 200 OK responses**: Possible misconfigured security rules
- **Time-based changes**: Rules might be too permissive during certain hours

---

## 📜 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

## ⚠️ Disclaimer

This tool is for **authorized security testing only**. Only use FireY on Firebase projects you own or have explicit permission to test. Unauthorized access to computer systems is illegal.

---

## 🌟 Star History

If you find FireY useful, please consider giving it a star! ⭐

---

## 📞 Support

- 🐛 **Issues**: [GitHub Issues](https://github.com/sirbugs/firey/issues)
- 💬 **Discussions**: [GitHub Discussions](https://github.com/sirbugs/firey/discussions)
- 📧 **Email**: security@yourproject.com

---

<div align="center">

**Made with ❤️ for Firebase Security**

[⬆ Back to Top](#-firey)

</div>
