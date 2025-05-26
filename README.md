# Automate Home

A Go-based home automation service that integrates with Tuya smart devices and provides a web API for controlling curtains and other IoT devices through scene automation.

## Features

- 🏠 **Home Automation**: Control smart home devices through HTTP API
- 🪟 **Curtain Control**: Automated curtain opening/closing via Tuya API
- 🌐 **Web Interface Integration**: Headless browser automation for gateway interaction
- 🔄 **Scene Management**: Predefined scenes for different home automation scenarios
- 🐳 **Docker Support**: Containerized deployment with automatic restarts
- 📊 **Real-time Monitoring**: Continuous gateway monitoring and device status checking

## Architecture

The service consists of:

1. **HTTP Server**: Listens on port 8080 for scene control requests
2. **Browser Automation**: Uses ChromeDP for web gateway interaction
3. **Tuya Integration**: Direct API calls to Tuya Cloud for device control
4. **Scene Controller**: Manages predefined automation scenarios

## Prerequisites

- Go 1.24.3 or later
- Docker and Docker Compose
- Tuya Cloud account with API credentials
- Home gateway/router with web interface

## Quick Start

### 1. Clone the Repository

```bash
git clone <repository-url>
cd automate-home
```

### 2. Environment Configuration

Copy the example environment file and configure your settings:

```bash
cp example.env .env
```

Edit `.env` with your specific configuration:

```env
TZ=Asia/Bangkok
HOST=http://your-gateway-host:port
USERNAME=your-gateway-username
PASS=your-gateway-password
ACCESS_ID=your-tuya-access-id
ACCESS_KEY=your-tuya-access-key
DEVICE_ID_1=your-first-device-id
DEVICE_ID_2=your-second-device-id
```

### 3. Run with Docker Compose

```bash
docker-compose up --build -d
```

The service will be available at `http://localhost:8090`

### 4. Manual Run (Development)

```bash
# Install dependencies
go mod download

# Run the application
cd src && go run .
```

## API Usage

### Play Scene

Send a POST request to trigger automation scenes:

```bash
curl -X POST http://localhost:8090/play-scene \
  -H "Content-Type: application/json" \
  -d '{"scene": 1}'
```

**Available Scenes:**

- `Scene 1`: Close all curtains
- `Scene 2`: Open all curtains  
- `Scene 3`: Close all curtains (alternative)

### Response Format

```json
{
  "scene": 1
}
```

## Configuration

### Environment Variables

| Variable | Description | Required |
|----------|-------------|----------|
| `TZ` | Timezone for the application | Yes |
| `HOST` | Gateway/router web interface URL | Yes |
| `USERNAME` | Gateway authentication username | Yes |
| `PASS` | Gateway authentication password | Yes |
| `ACCESS_ID` | Tuya Cloud API Access ID | Yes |
| `ACCESS_KEY` | Tuya Cloud API Access Key | Yes |
| `DEVICE_ID_1` | First Tuya device ID (curtain) | Yes |
| `DEVICE_ID_2` | Second Tuya device ID (curtain) | Yes |

### Tuya Device Setup

1. Register your devices in the Tuya Smart app
2. Create a Tuya Cloud project at [Tuya IoT Platform](https://iot.tuya.com)
3. Link your devices to the cloud project
4. Obtain the Access ID, Access Key, and Device IDs
5. Configure the environment variables accordingly

## Development

### Project Structure

```
.
├── src/
│   ├── main.go           # Main application and HTTP server
│   └── tuya_client.go    # Tuya API client implementation
├── docker-compose.yml    # Docker Compose configuration
├── Dockerfile           # Container build instructions
├── go.mod              # Go module dependencies
├── go.sum              # Dependency checksums
├── .env                # Environment configuration (not in repo)
└── example.env         # Environment template
```

### Building

```bash
# Build for current platform
go build -o automate-home ./src

# Build for Linux (for Docker)
CGO_ENABLED=0 GOOS=linux go build -o automate-home ./src
```

### Dependencies

- **chromedp**: Headless browser automation
- **godotenv**: Environment variable management
- **Tuya Cloud API**: IoT device control

## Docker Deployment

The application includes two services:

1. **app**: Main application container
2. **restarter**: Daily restart service for reliability

### Container Features

- Based on `chromedp/headless-shell` for browser automation
- Multi-stage build for optimized image size
- Automatic daily restarts for long-running stability
- Volume mounting for environment configuration

## Monitoring and Logs

View application logs:

```bash
# Docker Compose logs
docker-compose logs -f app

# Individual container logs
docker logs -f automate-home
```

## Troubleshooting

### Common Issues

1. **Gateway Connection Failed**
   - Verify `HOST`, `USERNAME`, and `PASS` in `.env`
   - Ensure gateway is accessible from container network

2. **Tuya API Authentication Failed**
   - Check `ACCESS_ID` and `ACCESS_KEY` credentials
   - Verify device IDs are correct and linked to your project

3. **Device Commands Not Working**
   - Confirm devices are online in Tuya app
   - Check device permissions in Tuya Cloud project

### Debug Mode

For development debugging, you can run with verbose logging:

```bash
cd src && go run . -v
```

## Security Considerations

- Store credentials securely in `.env` file
- Never commit `.env` to version control
- Use network restrictions for production deployments
- Regularly rotate Tuya API credentials

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Test thoroughly
5. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Support

For issues and questions:

1. Check the troubleshooting section
2. Review application logs
3. Open an issue on GitHub with detailed information

---

**Note**: This automation system is designed for personal use. Ensure all devices and credentials are properly secured in production environments.
