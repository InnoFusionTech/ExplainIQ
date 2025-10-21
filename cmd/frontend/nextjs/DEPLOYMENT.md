# ExplainIQ Frontend Deployment Guide

## Quick Start

### Prerequisites
- Node.js 18+
- npm or yarn
- Running ExplainIQ orchestrator service

### Local Development

1. **Install dependencies:**
```bash
cd cmd/frontend/nextjs
npm install
```

2. **Set environment variables:**
```bash
# Create .env.local file
echo "ORCHESTRATOR_URL=http://localhost:8080" > .env.local
```

3. **Start development server:**
```bash
npm run dev
```

4. **Open browser:**
Navigate to [http://localhost:3000](http://localhost:3000)

### Production Build

1. **Build the application:**
```bash
npm run build
```

2. **Start production server:**
```bash
npm start
```

### Docker Deployment

1. **Build Docker image:**
```bash
docker build -t explainiq-frontend .
```

2. **Run container:**
```bash
docker run -p 3000:3000 -e ORCHESTRATOR_URL=http://host.docker.internal:8080 explainiq-frontend
```

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `ORCHESTRATOR_URL` | Backend orchestrator service URL | `http://localhost:8080` |
| `NEXT_TELEMETRY_DISABLED` | Disable Next.js telemetry | `1` |

### Testing

```bash
# Run tests
npm test

# Run tests in watch mode
npm run test:watch

# Run linter
npm run lint
```

### Deployment Platforms

#### Vercel (Recommended)
1. Connect your repository to Vercel
2. Set environment variables in Vercel dashboard
3. Deploy automatically on push

#### Netlify
1. Connect repository to Netlify
2. Set build command: `npm run build`
3. Set publish directory: `.next`
4. Set environment variables

#### Docker
Use the provided Dockerfile for containerized deployment.

### Troubleshooting

#### Common Issues

1. **Build fails with TypeScript errors:**
   - Run `npm run lint` to check for issues
   - Ensure all types are properly defined

2. **SSE connection fails:**
   - Check `ORCHESTRATOR_URL` environment variable
   - Ensure orchestrator service is running
   - Check CORS settings

3. **Images not loading:**
   - Verify GCS bucket configuration
   - Check image URLs in browser network tab
   - Ensure proper CORS headers on GCS

#### Performance Optimization

1. **Enable compression:**
   - Use gzip/brotli compression
   - Optimize images with Next.js Image component

2. **Caching:**
   - Set appropriate cache headers
   - Use CDN for static assets

3. **Bundle optimization:**
   - Analyze bundle size with `npm run build`
   - Use dynamic imports for large components

### Monitoring

- Use Next.js built-in analytics
- Monitor Core Web Vitals
- Track SSE connection health
- Monitor API response times

### Security

- Use HTTPS in production
- Set secure headers
- Validate all inputs
- Use environment variables for secrets



