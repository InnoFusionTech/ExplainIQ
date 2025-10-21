# Environment Setup Guide

This guide explains how to set up your environment variables and secrets for the ExplainIQ Agent system.

## 🔐 Environment Variables

### 1. Create Your .env File

The system uses a `.env` file to store all environment variables and secrets. A template has been created for you.

**Edit the `.env` file and update these key variables:**

```bash
# AI MODEL CONFIGURATION
GEMINI_API_KEY=your-actual-api-key-here

# EXPLAINIQ CORE CONFIGURATION  
EXPLAINIQ_PROJECT_ID=your-project-id
EXPLAINIQ_REGION=us-central1

# AUTHENTICATION
JWT_SECRET=your-super-secret-jwt-key-change-this-in-production
```

### 2. Get Your Gemini API Key

1. Go to [Google AI Studio](https://makersuite.google.com/app/apikey)
2. Sign in with your Google account
3. Click "Create API Key"
4. Copy the key (starts with `AIzaSyC...`)
5. Replace `your-actual-api-key-here` in your `.env` file

### 3. Generate JWT Secret

Generate a strong random string for JWT authentication:

```bash
# PowerShell
[System.Web.Security.Membership]::GeneratePassword(32, 0)

# Or use online generator
# https://www.allkeysgenerator.com/Random/Security-Encryption-Key-Generator.aspx
```

## 🚀 Starting Services

### Option 1: Use the Startup Script (Recommended)

```powershell
.\start-services.ps1
```

This script will:
- Load environment variables from `.env`
- Verify your API key is set
- Start all services with real AI
- Show you the service URLs

### Option 2: Manual Startup

1. **Load environment variables:**
   ```powershell
   .\load-env.ps1
   ```

2. **Start services individually:**
   ```powershell
   # Start Orchestrator
   .\orchestrator.exe
   
   # Start Summarizer with Gemini AI
   .\cmd\agent-summarizer\gemini_summarizer.exe
   
   # Start other services...
   ```

## 🧪 Testing

### Test AI Integration

```powershell
.\test-ai.ps1
```

This will test if your Gemini API key is working correctly.

### Manual API Test

```powershell
$body = @{
    session_id = "test-123"
    inputs = @{
        topic = "quantum computing"
        context = "Quantum computing uses quantum mechanical phenomena"
    }
} | ConvertTo-Json -Depth 3

Invoke-WebRequest -Uri "http://localhost:8081/task" -Method POST -ContentType "application/json" -Body $body
```

## 🛑 Stopping Services

```powershell
.\stop-services.ps1
```

## 📋 Service URLs

Once started, services are available at:

- **Orchestrator**: http://localhost:8080
- **Summarizer**: http://localhost:8081 (with real Gemini AI)
- **Explainer**: http://localhost:8082
- **Critic**: http://localhost:8083
- **Visualizer**: http://localhost:8084
- **Frontend**: http://localhost:8085

## 🔒 Security Notes

- The `.env` file is in `.gitignore` and won't be committed
- Never share your API keys
- Use different keys for development and production
- For production, use Google Cloud Secret Manager

## 🐛 Troubleshooting

### "API Key Not Set" Error
- Make sure you've edited the `.env` file
- Verify the API key is correct
- Restart the services after changing the `.env` file

### "Service Not Responding" Error
- Check if the service is running: `netstat -an | findstr ":808"`
- Restart the service
- Check the service logs

### "Mock Responses" Instead of Real AI
- Verify `GEMINI_API_KEY` is set correctly
- Make sure you're using `gemini_summarizer.exe` (not `simple_summarizer.exe`)
- Check the service logs for API errors

## 📁 File Structure

```
├── .env                    # Your environment variables (edit this)
├── .env.local             # Local overrides (optional)
├── start-services.ps1     # Start all services
├── stop-services.ps1      # Stop all services  
├── test-ai.ps1           # Test AI integration
└── load-env.ps1          # Load environment variables
```

## 🎯 Next Steps

1. **Edit `.env`** with your API key
2. **Run `.\start-services.ps1`**
3. **Test with `.\test-ai.ps1`**
4. **Open http://localhost:8085** in your browser
5. **Enjoy your AI-powered learning platform!**

