#!/bin/bash

# ExplainIQ Frontend Deployment Script

set -e

echo "🚀 Starting ExplainIQ Frontend deployment..."

# Check if Node.js is installed
if ! command -v node &> /dev/null; then
    echo "❌ Node.js is not installed. Please install Node.js 18+ first."
    exit 1
fi

# Check Node.js version
NODE_VERSION=$(node -v | cut -d'v' -f2 | cut -d'.' -f1)
if [ "$NODE_VERSION" -lt 18 ]; then
    echo "❌ Node.js version 18+ is required. Current version: $(node -v)"
    exit 1
fi

echo "✅ Node.js version: $(node -v)"

# Install dependencies
echo "📦 Installing dependencies..."
npm ci

# Run linting
echo "🔍 Running linter..."
npm run lint

# Run tests
echo "🧪 Running tests..."
npm test

# Build the application
echo "🏗️  Building application..."
npm run build

echo "✅ Build completed successfully!"
echo ""
echo "To start the application:"
echo "  npm start"
echo ""
echo "To run in development mode:"
echo "  npm run dev"
echo ""
echo "Environment variables needed:"
echo "  ORCHESTRATOR_URL=http://localhost:8080"



