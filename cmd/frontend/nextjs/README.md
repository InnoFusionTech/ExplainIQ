# ExplainIQ Frontend

A modern, mobile-friendly Next.js frontend for the ExplainIQ learning platform.

## Features

- **Topic Input Form**: Simple, clean form to input learning topics
- **Real-time Progress**: Server-Sent Events (SSE) for live updates during learning session
- **Vertical Timeline**: Visual progress tracking with step badges and status indicators
- **Final Results**: Structured display of OG lesson sections and generated visualizations
- **PDF Generation**: Download lessons as professionally formatted PDFs
- **Cloud Storage**: Automatic PDF storage and signed URL generation
- **Mobile-First Design**: Responsive design optimized for all device sizes
- **Minimal Dependencies**: Lightweight with only essential packages

## Tech Stack

- **Next.js 14**: React framework with App Router
- **TypeScript**: Type-safe development
- **Tailwind CSS**: Utility-first CSS framework
- **Server-Sent Events**: Real-time communication with backend
- **Puppeteer**: PDF generation from HTML
- **Google Cloud Storage**: PDF storage and signed URLs

## Getting Started

### Prerequisites

- Node.js 18+ 
- npm or yarn
- Running ExplainIQ orchestrator service
- Google Cloud Storage bucket (for PDF storage)
- Google Cloud credentials (for PDF uploads)

### Installation

1. Install dependencies:
```bash
npm install
```

2. Set environment variables:
```bash
# Create .env.local file
ORCHESTRATOR_URL=http://localhost:8080
GCS_BUCKET=your-pdf-bucket
GCS_PROJECT_ID=your-gcp-project-id
```

3. Run the development server:
```bash
npm run dev
```

4. Open [http://localhost:3000](http://localhost:3000) in your browser.

### Building for Production

```bash
npm run build
npm start
```

## Project Structure

```
nextjs/
├── pages/
│   ├── api/
│   │   └── sessions/
│   │       ├── index.ts          # POST /api/sessions
│   │       └── [id]/
│   │           └── run.ts        # GET /api/sessions/[id]/run (SSE)
│   ├── _app.tsx                  # App wrapper
│   └── index.tsx                 # Main page
├── styles/
│   └── globals.css               # Global styles with Tailwind
├── types/
│   └── index.ts                  # TypeScript type definitions
├── package.json
├── tailwind.config.js
├── tsconfig.json
└── next.config.js
```

## API Endpoints

### Frontend API Routes

- `POST /api/sessions` - Create a new learning session
- `GET /api/sessions/[id]/run` - SSE stream for session progress
- `POST /api/sessions/[id]/pdf` - Generate and download PDF of lesson

### Backend Integration

The frontend communicates with the ExplainIQ orchestrator service:

- **Session Creation**: `POST /api/v1/sessions`
- **Progress Updates**: `GET /api/v1/sessions/{id}/run` (SSE)

## Components

### Main Page (`pages/index.tsx`)

- Topic input form
- Real-time progress timeline
- Final results display with lesson sections and images

### Timeline Component

Visual progress tracking with:
- Step badges (Summarizer, Explainer, Visualizer, Critic)
- Status indicators (pending, running, completed, failed)
- Duration tracking
- Error handling

### Final Results

Structured display of:
- **Big Picture**: High-level overview
- **Metaphor**: Analogical explanation
- **Core Mechanism**: Technical details
- **Toy Example**: Code examples
- **Memory Hook**: Mnemonic device
- **Real Life**: Practical applications
- **Best Practices**: Implementation guidelines
- **Visualizations**: Generated diagrams with captions
- **PDF Download**: Professional PDF export with all content

## Styling

Uses Tailwind CSS with custom components:

- `.btn-primary` - Primary action buttons
- `.card` - Content containers
- `.timeline-item` - Progress timeline items
- `.section-*` - Lesson section styling

## Mobile Responsiveness

- Mobile-first design approach
- Responsive grid layouts
- Touch-friendly interface
- Optimized for small screens

## Environment Variables

- `ORCHESTRATOR_URL`: Backend orchestrator service URL (default: http://localhost:8080)
- `GCS_BUCKET`: Google Cloud Storage bucket for PDF storage
- `GCS_PROJECT_ID`: Google Cloud Project ID for authentication

## Development

### Code Style

- TypeScript for type safety
- ESLint for code quality
- Prettier for formatting (optional)

### Testing

```bash
npm run lint
```

## Deployment

The frontend can be deployed to:

- **Vercel** (recommended for Next.js)
- **Netlify**
- **Docker** container
- Any static hosting service

### Docker Deployment

```dockerfile
FROM node:18-alpine
WORKDIR /app
COPY package*.json ./
RUN npm ci --only=production
COPY . .
RUN npm run build
EXPOSE 3000
CMD ["npm", "start"]
```

## Browser Support

- Chrome 90+
- Firefox 88+
- Safari 14+
- Edge 90+

## Performance

- Server-side rendering (SSR)
- Automatic code splitting
- Image optimization
- Minimal bundle size
- Fast loading times

## Accessibility

- Semantic HTML structure
- ARIA labels and roles
- Keyboard navigation support
- Screen reader compatibility
- High contrast support
