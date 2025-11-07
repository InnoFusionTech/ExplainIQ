import { NextApiRequest, NextApiResponse } from 'next';
import puppeteer from 'puppeteer';
import { Storage } from '@google-cloud/storage';
import { PDFResponse, OGLesson, ImageRef } from '../../../../types';
import { getOrchestratorURL } from '../../../../utils/orchestrator';
const GCS_BUCKET = process.env.GCS_BUCKET || 'explainiq-pdfs';
const GCS_PROJECT_ID = process.env.GCS_PROJECT_ID || '';

// Initialize Google Cloud Storage
const storage = new Storage({
  projectId: GCS_PROJECT_ID,
});

export default async function handler(
  req: NextApiRequest,
  res: NextApiResponse<PDFResponse | { error: string }>
) {
  if (req.method !== 'POST') {
    return res.status(405).json({ error: 'Method not allowed' });
  }

  const { id: sessionId } = req.query;

  if (!sessionId || typeof sessionId !== 'string') {
    return res.status(400).json({ error: 'Session ID is required' });
  }

  try {
    // Fetch session data from orchestrator
    const ORCHESTRATOR_URL = getOrchestratorURL();
    console.log(`[PDF Generation] Fetching from: ${ORCHESTRATOR_URL}/api/sessions/result?id=${sessionId}`);
    
    const sessionResponse = await fetch(`${ORCHESTRATOR_URL}/api/sessions/result?id=${sessionId}`);
    
    if (!sessionResponse.ok) {
      throw new Error(`Failed to fetch session data: ${sessionResponse.statusText}`);
    }

    const sessionData = await sessionResponse.json();
    
    if (!sessionData.artifacts || !sessionData.artifacts.lesson) {
      throw new Error('Session data is incomplete - lesson not found');
    }

    // Parse lesson data
    const lesson: OGLesson = JSON.parse(sessionData.artifacts.lesson);
    const images: ImageRef[] = sessionData.artifacts.images ? JSON.parse(sessionData.artifacts.images) : [];
    const topic = sessionData.topic || 'Learning Topic';

    // Generate PDF
    const pdfBuffer = await generatePDF(lesson, images, topic, sessionId);

    // Upload to Google Cloud Storage
    const filename = `lesson-${sessionId}-${Date.now()}.pdf`;
    const bucket = storage.bucket(GCS_BUCKET);
    const file = bucket.file(`sessions/${sessionId}/${filename}`);

    await file.save(pdfBuffer, {
      metadata: {
        contentType: 'application/pdf',
        cacheControl: 'public, max-age=31536000',
      },
    });

    // Generate signed URL (valid for 1 hour)
    const [signedUrl] = await file.getSignedUrl({
      action: 'read',
      expires: Date.now() + 60 * 60 * 1000, // 1 hour
    });

    // Get file size
    const [metadata] = await file.getMetadata();
    const size = typeof metadata.size === 'string' 
      ? parseInt(metadata.size) 
      : metadata.size || 0;

    const response: PDFResponse = {
      pdf_url: signedUrl,
      filename,
      size,
      created_at: new Date().toISOString(),
    };

    res.status(200).json(response);
  } catch (error) {
    console.error('PDF generation error:', error);
    res.status(500).json({
      error: error instanceof Error ? error.message : 'PDF generation failed',
    });
  }
}

async function generatePDF(lesson: OGLesson, images: ImageRef[], topic: string, sessionId: string): Promise<Buffer> {
  const launchOptions: any = {
    headless: true,
    args: [
      '--no-sandbox',
      '--disable-setuid-sandbox',
      '--disable-dev-shm-usage',
      '--disable-accelerated-2d-canvas',
      '--no-first-run',
      '--no-zygote',
      '--disable-gpu',
    ],
  };

  // Use system Chromium in production (Docker)
  if (process.env.PUPPETEER_EXECUTABLE_PATH) {
    launchOptions.executablePath = process.env.PUPPETEER_EXECUTABLE_PATH;
  }

  const browser = await puppeteer.launch(launchOptions);

  try {
    const page = await browser.newPage();
    
    // Set viewport for consistent rendering
    await page.setViewport({ width: 1200, height: 800 });

    // Generate HTML content
    const htmlContent = generateHTMLContent(lesson, images, topic, sessionId);

    // Set content
    await page.setContent(htmlContent, { waitUntil: 'networkidle0' });

    // Generate PDF
    const pdfBuffer = await page.pdf({
      format: 'A4',
      printBackground: true,
      margin: {
        top: '20mm',
        right: '15mm',
        bottom: '20mm',
        left: '15mm',
      },
      displayHeaderFooter: true,
      headerTemplate: `
        <div style="font-size: 10px; text-align: center; width: 100%; color: #666;">
          <span>ExplainIQ Learning Experience</span>
        </div>
      `,
      footerTemplate: `
        <div style="font-size: 10px; text-align: center; width: 100%; color: #666;">
          <span>Generated on ${new Date().toLocaleDateString()} | Page <span class="pageNumber"></span> of <span class="totalPages"></span></span>
        </div>
      `,
    });

    return pdfBuffer;
  } finally {
    await browser.close();
  }
}

function generateHTMLContent(lesson: OGLesson, images: ImageRef[], topic: string, sessionId: string): string {
  const sections = [
    { key: 'big_picture', title: 'Big Picture', content: lesson.big_picture, color: '#3b82f6' },
    { key: 'metaphor', title: 'Metaphor', content: lesson.metaphor, color: '#10b981' },
    { key: 'core_mechanism', title: 'Core Mechanism', content: lesson.core_mechanism, color: '#8b5cf6' },
    { key: 'toy_example_code', title: 'Toy Example', content: lesson.toy_example_code, color: '#f59e0b', isCode: true },
    { key: 'memory_hook', title: 'Memory Hook', content: lesson.memory_hook, color: '#ec4899' },
    { key: 'real_life', title: 'Real Life', content: lesson.real_life, color: '#6366f1' },
    { key: 'best_practices', title: 'Best Practices', content: lesson.best_practices, color: '#eab308' },
  ];

  const imagesHTML = images.map((image, index) => `
    <div style="margin: 20px 0; text-align: center;">
      <img src="${image.url}" alt="${image.alt_text}" style="max-width: 100%; height: auto; border: 1px solid #e5e7eb; border-radius: 8px;" />
      <p style="font-size: 12px; color: #6b7280; margin-top: 8px;">${image.caption}</p>
    </div>
  `).join('');

  const sectionsHTML = sections.map(section => `
    <div style="margin: 30px 0; border-left: 4px solid ${section.color}; padding-left: 20px;">
      <h3 style="color: ${section.color}; font-size: 18px; font-weight: 600; margin: 0 0 10px 0;">${section.title}</h3>
      ${section.isCode ? `
        <pre style="background-color: #f3f4f6; padding: 15px; border-radius: 6px; font-size: 12px; overflow-x: auto; margin: 0;">${section.content}</pre>
      ` : `
        <p style="color: #374151; line-height: 1.6; margin: 0;">${section.content}</p>
      `}
    </div>
  `).join('');

  return `
    <!DOCTYPE html>
    <html lang="en">
    <head>
      <meta charset="UTF-8">
      <meta name="viewport" content="width=device-width, initial-scale=1.0">
      <title>ExplainIQ - ${topic}</title>
      <style>
        body {
          font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
          line-height: 1.6;
          color: #1f2937;
          margin: 0;
          padding: 0;
        }
        .header {
          text-align: center;
          margin-bottom: 40px;
          padding-bottom: 20px;
          border-bottom: 2px solid #e5e7eb;
        }
        .header h1 {
          color: #1f2937;
          font-size: 28px;
          font-weight: 700;
          margin: 0 0 10px 0;
        }
        .header p {
          color: #6b7280;
          font-size: 16px;
          margin: 0;
        }
        .content {
          max-width: 800px;
          margin: 0 auto;
        }
        .visualizations {
          margin: 40px 0;
        }
        .visualizations h3 {
          color: #1f2937;
          font-size: 20px;
          font-weight: 600;
          margin: 0 0 20px 0;
        }
        @media print {
          body { -webkit-print-color-adjust: exact; }
        }
      </style>
    </head>
    <body>
      <div class="content">
        <div class="header">
          <h1>ExplainIQ Learning Experience</h1>
          <p>${topic}</p>
        </div>
        
        ${sectionsHTML}
        
        ${images.length > 0 ? `
          <div class="visualizations">
            <h3>Visualizations</h3>
            ${imagesHTML}
          </div>
        ` : ''}
      </div>
    </body>
    </html>
  `;
}
