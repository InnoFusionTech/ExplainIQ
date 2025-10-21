# PDF Generation Guide

## Overview

The ExplainIQ frontend includes comprehensive PDF generation functionality that converts completed learning sessions into professionally formatted PDF documents. This feature uses Puppeteer for HTML-to-PDF conversion and Google Cloud Storage for secure file storage and distribution.

## Features

- **Professional Formatting**: Clean, print-ready PDF layout with proper typography
- **Complete Content**: Includes all lesson sections and visualizations
- **Cloud Storage**: Automatic upload to Google Cloud Storage with signed URLs
- **Security**: Time-limited signed URLs for secure access
- **Error Handling**: Comprehensive error handling and user feedback
- **Mobile Friendly**: Download button works seamlessly on all devices

## Architecture

### Components

1. **PDF API Endpoint** (`/api/sessions/[id]/pdf`)
   - Fetches session data from orchestrator
   - Generates HTML template
   - Converts to PDF using Puppeteer
   - Uploads to Google Cloud Storage
   - Returns signed URL

2. **HTML Template Generator**
   - Creates print-optimized HTML
   - Includes all lesson sections
   - Embeds visualizations
   - Professional styling

3. **Download Button** (LessonCard component)
   - Triggers PDF generation
   - Shows loading states
   - Handles errors gracefully
   - Initiates download

### Data Flow

```
User clicks "Download PDF"
    ↓
Frontend calls /api/sessions/{id}/pdf
    ↓
API fetches session data from orchestrator
    ↓
API generates HTML template
    ↓
Puppeteer converts HTML to PDF
    ↓
PDF uploaded to Google Cloud Storage
    ↓
Signed URL generated and returned
    ↓
Frontend triggers download
```

## API Reference

### POST /api/sessions/[id]/pdf

Generates a PDF from a completed learning session.

**Request:**
```http
POST /api/sessions/session-123/pdf
Content-Type: application/json
```

**Response:**
```json
{
  "pdf_url": "https://storage.googleapis.com/bucket/sessions/session-123/lesson-session-123-1234567890.pdf",
  "filename": "lesson-session-123-1234567890.pdf",
  "size": 1024000,
  "created_at": "2024-01-01T12:00:00Z"
}
```

**Error Response:**
```json
{
  "error": "Session data is incomplete - lesson not found"
}
```

## Configuration

### Environment Variables

```bash
# Google Cloud Storage
GCS_BUCKET=explainiq-pdfs
GCS_PROJECT_ID=your-gcp-project-id

# Puppeteer (optional, for Docker)
PUPPETEER_SKIP_CHROMIUM_DOWNLOAD=true
PUPPETEER_EXECUTABLE_PATH=/usr/bin/chromium-browser
```

### Google Cloud Setup

1. **Create Storage Bucket:**
```bash
gsutil mb gs://your-pdf-bucket
```

2. **Set Permissions:**
```bash
# Allow public read access to PDFs
gsutil iam ch allUsers:objectViewer gs://your-pdf-bucket
```

3. **Service Account:**
   - Create service account with Storage Admin role
   - Download JSON key file
   - Set `GOOGLE_APPLICATION_CREDENTIALS` environment variable

## PDF Template

### Structure

The generated PDF includes:

1. **Header**
   - ExplainIQ branding
   - Topic title
   - Generation timestamp

2. **Lesson Sections**
   - Big Picture (blue border)
   - Metaphor (green border)
   - Core Mechanism (purple border)
   - Toy Example (orange border, code formatting)
   - Memory Hook (pink border)
   - Real Life (indigo border)
   - Best Practices (yellow border)

3. **Visualizations**
   - Embedded images with captions
   - Proper scaling and layout

4. **Footer**
   - Page numbers
   - Generation date

### Styling

- **Typography**: System fonts for optimal readability
- **Colors**: Consistent with web interface
- **Layout**: Print-optimized margins and spacing
- **Images**: High-quality rendering with proper scaling

## Usage Examples

### Basic Usage

```typescript
// Frontend component
const handleDownloadPDF = async () => {
  const response = await fetch(`/api/sessions/${sessionId}/pdf`, {
    method: 'POST',
  });
  
  const pdfData = await response.json();
  
  // Trigger download
  const link = document.createElement('a');
  link.href = pdfData.pdf_url;
  link.download = pdfData.filename;
  link.click();
};
```

### Error Handling

```typescript
try {
  const response = await fetch(`/api/sessions/${sessionId}/pdf`, {
    method: 'POST',
  });
  
  if (!response.ok) {
    const errorData = await response.json();
    throw new Error(errorData.error);
  }
  
  // Handle success
} catch (error) {
  // Handle error
  console.error('PDF generation failed:', error);
}
```

## Performance Considerations

### Optimization

1. **Puppeteer Configuration**
   - Headless mode for server environments
   - Optimized launch arguments
   - Proper resource cleanup

2. **PDF Generation**
   - Efficient HTML template generation
   - Optimized image handling
   - Minimal memory usage

3. **Cloud Storage**
   - Efficient upload streaming
   - Proper metadata setting
   - Cache-friendly headers

### Caching

- PDFs are cached in Google Cloud Storage
- Signed URLs are valid for 1 hour
- Browser caching for repeated downloads

## Security

### Access Control

- PDFs are stored in private GCS buckets
- Signed URLs provide time-limited access
- No direct public access to PDF files

### Data Protection

- Session data is fetched securely from orchestrator
- No sensitive data stored in PDFs
- Proper error handling prevents data leaks

## Troubleshooting

### Common Issues

1. **PDF Generation Fails**
   - Check Puppeteer installation
   - Verify Chromium dependencies
   - Check memory limits

2. **Upload Fails**
   - Verify GCS credentials
   - Check bucket permissions
   - Verify network connectivity

3. **Download Issues**
   - Check signed URL validity
   - Verify CORS settings
   - Check browser compatibility

### Debug Mode

Enable debug logging:

```bash
DEBUG=puppeteer:*
```

### Health Checks

Test PDF generation:

```bash
curl -X POST http://localhost:3000/api/sessions/test-session/pdf
```

## Monitoring

### Metrics to Track

- PDF generation success rate
- Generation time
- File sizes
- Download counts
- Error rates

### Logging

Key events to log:
- PDF generation start/completion
- Upload success/failure
- Download requests
- Error conditions

## Future Enhancements

### Planned Features

1. **Custom Templates**
   - User-selectable PDF templates
   - Brand customization
   - Layout options

2. **Batch Generation**
   - Multiple session PDFs
   - Bulk download
   - Archive creation

3. **Advanced Formatting**
   - Table of contents
   - Index generation
   - Custom styling

4. **Integration**
   - Email delivery
   - Cloud storage sync
   - API webhooks



