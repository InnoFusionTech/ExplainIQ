import { createMocks } from 'node-mocks-http';
import handler from '../pages/api/sessions/[id]/pdf';

// Mock puppeteer
jest.mock('puppeteer', () => ({
  launch: jest.fn().mockResolvedValue({
    newPage: jest.fn().mockResolvedValue({
      setViewport: jest.fn(),
      setContent: jest.fn(),
      pdf: jest.fn().mockResolvedValue(Buffer.from('mock-pdf-content')),
    }),
    close: jest.fn(),
  }),
}));

// Mock Google Cloud Storage
jest.mock('@google-cloud/storage', () => ({
  Storage: jest.fn().mockImplementation(() => ({
    bucket: jest.fn().mockReturnValue({
      file: jest.fn().mockReturnValue({
        save: jest.fn(),
        getSignedUrl: jest.fn().mockResolvedValue(['https://signed-url.com']),
        getMetadata: jest.fn().mockResolvedValue([{ size: '1024' }]),
      }),
    }),
  })),
}));

// Mock fetch
global.fetch = jest.fn();

describe('/api/sessions/[id]/pdf', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('should generate PDF successfully', async () => {
    // Mock orchestrator response
    (global.fetch as jest.Mock).mockResolvedValueOnce({
      ok: true,
      json: async () => ({
        topic: 'Machine Learning',
        artifacts: {
          lesson: JSON.stringify({
            big_picture: 'ML is a subset of AI',
            metaphor: 'Like teaching a child',
            core_mechanism: 'Algorithms find patterns',
            toy_example_code: 'model.fit(X, y)',
            memory_hook: 'ML = More Learning',
            real_life: 'Used in recommendations',
            best_practices: 'Clean your data',
          }),
          images: JSON.stringify([]),
        },
      }),
    });

    const { req, res } = createMocks({
      method: 'POST',
      query: { id: 'test-session-id' },
    });

    await handler(req, res);

    expect(res._getStatusCode()).toBe(200);
    const data = JSON.parse(res._getData());
    expect(data).toHaveProperty('pdf_url');
    expect(data).toHaveProperty('filename');
    expect(data).toHaveProperty('size');
    expect(data).toHaveProperty('created_at');
  });

  it('should return 405 for non-POST requests', async () => {
    const { req, res } = createMocks({
      method: 'GET',
      query: { id: 'test-session-id' },
    });

    await handler(req, res);

    expect(res._getStatusCode()).toBe(405);
    const data = JSON.parse(res._getData());
    expect(data).toHaveProperty('error', 'Method not allowed');
  });

  it('should return 400 for missing session ID', async () => {
    const { req, res } = createMocks({
      method: 'POST',
      query: {},
    });

    await handler(req, res);

    expect(res._getStatusCode()).toBe(400);
    const data = JSON.parse(res._getData());
    expect(data).toHaveProperty('error', 'Session ID is required');
  });

  it('should handle orchestrator fetch error', async () => {
    (global.fetch as jest.Mock).mockResolvedValueOnce({
      ok: false,
      statusText: 'Not Found',
    });

    const { req, res } = createMocks({
      method: 'POST',
      query: { id: 'test-session-id' },
    });

    await handler(req, res);

    expect(res._getStatusCode()).toBe(500);
    const data = JSON.parse(res._getData());
    expect(data).toHaveProperty('error');
    expect(data.error).toContain('Failed to fetch session data');
  });

  it('should handle incomplete session data', async () => {
    (global.fetch as jest.Mock).mockResolvedValueOnce({
      ok: true,
      json: async () => ({
        topic: 'Machine Learning',
        artifacts: {}, // Missing lesson
      }),
    });

    const { req, res } = createMocks({
      method: 'POST',
      query: { id: 'test-session-id' },
    });

    await handler(req, res);

    expect(res._getStatusCode()).toBe(500);
    const data = JSON.parse(res._getData());
    expect(data).toHaveProperty('error');
    expect(data.error).toContain('Session data is incomplete');
  });
});



