# AI Model Setup Guide

## Current Status
✅ **System is running with mock responses**
❌ **No real AI models connected yet**

## What's Working
- All services are running (orchestrator, agents, frontend)
- Mock responses are being generated
- Frontend UI is functional
- API endpoints are working

## To Connect Real AI Models

### 1. Google Gemini (Recommended)

#### Get API Key:
1. Go to [Google AI Studio](https://makersuite.google.com/app/apikey)
2. Create a new API key
3. Copy the key

#### Set Environment Variable:
```bash
# Windows PowerShell
$env:GEMINI_API_KEY="your-api-key-here"

# Windows CMD
set GEMINI_API_KEY=your-api-key-here

# Linux/Mac
export GEMINI_API_KEY="your-api-key-here"
```

#### Restart Services:
```bash
# Stop current services
Get-Process | Where-Object {$_.ProcessName -like "*simple*"} | Stop-Process -Force

# Start Gemini-enabled services
cd "C:\App\ExplainIQ Agent\cmd\agent-summarizer"
.\gemini_summarizer.exe
```

### 2. Test Real AI Connection

```powershell
# Test with real AI
$body = @{
    session_id = "test-123"
    inputs = @{
        topic = "quantum computing"
        context = "Quantum computing uses quantum mechanical phenomena"
    }
} | ConvertTo-Json -Depth 3

Invoke-WebRequest -Uri "http://localhost:8081/task" -Method POST -ContentType "application/json" -Body $body
```

### 3. Expected Behavior

**Without API Key:**
- Returns mock/hardcoded responses
- Works but not real AI

**With API Key:**
- Makes real calls to Gemini API
- Returns AI-generated content
- More intelligent and contextual responses

## Current Services Status

| Service | Port | Status | AI Model |
|---------|------|--------|----------|
| Orchestrator | 8080 | ✅ Running | N/A |
| Summarizer | 8081 | ✅ Running | Mock/Gemini |
| Explainer | 8082 | ✅ Running | Mock |
| Critic | 8083 | ✅ Running | Mock |
| Visualizer | 8084 | ✅ Running | Mock |
| Frontend | 3000 | ✅ Running | N/A |

## Next Steps

1. **Get Gemini API Key** (free tier available)
2. **Set environment variable**
3. **Restart services**
4. **Test with real AI responses**

## Alternative AI Models

The system can be extended to support:
- OpenAI GPT models
- Anthropic Claude
- Local models (Ollama, etc.)
- Azure OpenAI

## Cost Considerations

- **Gemini**: Free tier available (60 requests/minute)
- **Mock responses**: No cost, limited intelligence
- **Real AI**: More intelligent, may have usage costs

## Troubleshooting

### "No API Key" Error
- Set GEMINI_API_KEY environment variable
- Restart the service

### "API Request Failed"
- Check API key validity
- Check internet connection
- Verify API quota limits

### "JSON Parse Error"
- AI response format issue
- Falls back to mock response
- Check Gemini API documentation



