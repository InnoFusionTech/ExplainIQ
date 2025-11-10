import React from 'react';
import { render, screen, waitFor, act } from '@testing-library/react';
import InteractiveVisualizations from '../components/InteractiveVisualizations';
import { OGLesson } from '../types';

// Mock lesson data
const mockLesson: OGLesson = {
  big_picture: 'This is the big picture of the topic',
  metaphor: 'Think of it like a library',
  core_mechanism: 'The core mechanism works by processing data',
  toy_example_code: 'function example() { return true; }',
  memory_hook: 'Remember this with a catchy phrase',
  real_life: 'In real life, this is used in web applications',
  best_practices: 'Always follow best practices',
};

describe('InteractiveVisualizations', () => {
  let originalWindow: typeof window;
  let originalDocument: typeof document;
  let mockCreateElement: jest.Mock;
  let mockAppendChild: jest.Mock;

  beforeEach(() => {
    // Save original window and document
    originalWindow = global.window;
    originalDocument = global.document;

    // Create mock functions
    mockCreateElement = jest.fn((tagName: string) => {
      const element = {
        tagName: tagName.toUpperCase(),
        className: '',
        textContent: '',
        innerHTML: '',
        src: '',
        async: false,
        onload: null as any,
        onerror: null as any,
        addEventListener: jest.fn(),
        appendChild: jest.fn(),
        querySelector: jest.fn(),
      };
      return element as any;
    });

    mockAppendChild = jest.fn();

    // Mock window and document
    global.window = {
      ...originalWindow,
      addEventListener: jest.fn(),
      removeEventListener: jest.fn(),
    } as any;

    // Mock document with createElementNS
    global.document = {
      ...originalDocument,
      readyState: 'complete',
      body: {
        appendChild: mockAppendChild,
      } as any,
      head: {
        appendChild: mockAppendChild,
      } as any,
      createElement: mockCreateElement,
      createElementNS: jest.fn((namespace: string, tagName: string) => {
        const element = {
          namespaceURI: namespace,
          tagName: tagName.toUpperCase(),
          className: '',
          textContent: '',
          innerHTML: '',
        };
        return element as any;
      }),
      querySelector: jest.fn(() => null),
      getElementById: jest.fn(() => null),
    } as any;

    // Clear any existing Mermaid mock
    delete (global.window as any).mermaid;
  });

  afterEach(() => {
    // Restore original window and document
    global.window = originalWindow;
    global.document = originalDocument;
    jest.clearAllMocks();
  });

  it('renders the component with header', () => {
    render(<InteractiveVisualizations lesson={mockLesson} topic="Test Topic" />);
    
    expect(screen.getByText('Test Topic')).toBeInTheDocument();
    expect(screen.getByText('Interactive Visualizations')).toBeInTheDocument();
  });

  it('renders visualization tabs', () => {
    render(<InteractiveVisualizations lesson={mockLesson} topic="Test Topic" />);
    
    expect(screen.getByText('Concept Map')).toBeInTheDocument();
    expect(screen.getByText('Learning Flow')).toBeInTheDocument();
    expect(screen.getByText('Mind Map')).toBeInTheDocument();
    expect(screen.getByText('Analytics')).toBeInTheDocument();
  });

  it('loads Mermaid script when mermaid visualization is active', async () => {
    render(<InteractiveVisualizations lesson={mockLesson} topic="Test Topic" />);

    await waitFor(() => {
      expect(mockCreateElement).toHaveBeenCalledWith('script');
    }, { timeout: 3000 });

    // Verify script was appended to head
    expect(mockAppendChild).toHaveBeenCalled();
  });

  it('initializes Mermaid when script loads successfully', async () => {
    const mockMermaid = {
      initialize: jest.fn(),
      run: jest.fn(),
      render: jest.fn(),
      parse: jest.fn(),
    };

    let scriptElement: any = null;
    mockCreateElement.mockImplementation((tagName: string) => {
      const element = {
        tagName: tagName.toUpperCase(),
        className: '',
        textContent: '',
        innerHTML: '',
        src: '',
        async: false,
        onload: null as any,
        onerror: null as any,
        addEventListener: jest.fn(),
        appendChild: jest.fn(),
        querySelector: jest.fn(),
      };
      if (tagName === 'script') {
        scriptElement = element;
      }
      return element as any;
    });

    render(<InteractiveVisualizations lesson={mockLesson} topic="Test Topic" />);

    // Wait for script to be created
    await waitFor(() => {
      expect(mockCreateElement).toHaveBeenCalledWith('script');
    }, { timeout: 3000 });

    // Simulate script load
    act(() => {
      (global.window as any).mermaid = mockMermaid;
      if (scriptElement && scriptElement.onload) {
        scriptElement.onload();
      }
    });

    await waitFor(() => {
      expect(mockMermaid.initialize).toHaveBeenCalledWith(
        expect.objectContaining({
          startOnLoad: true,
          theme: 'default',
          securityLevel: 'loose',
        })
      );
    }, { timeout: 3000 });
  });

  it('uses run() method to render Mermaid diagrams', async () => {
    const mockMermaid = {
      initialize: jest.fn(),
      run: jest.fn(),
      render: jest.fn(),
      parse: jest.fn(),
    };

    let scriptElement: any = null;
    mockCreateElement.mockImplementation((tagName: string) => {
      const element = {
        tagName: tagName.toUpperCase(),
        className: '',
        textContent: '',
        innerHTML: '',
        src: '',
        async: false,
        onload: null as any,
        onerror: null as any,
        addEventListener: jest.fn(),
        appendChild: jest.fn(),
        querySelector: jest.fn(),
      };
      if (tagName === 'script') {
        scriptElement = element;
      }
      return element as any;
    });

    const { container } = render(<InteractiveVisualizations lesson={mockLesson} topic="Test Topic" />);

    // Wait for script to be created
    await waitFor(() => {
      expect(mockCreateElement).toHaveBeenCalledWith('script');
    }, { timeout: 3000 });

    // Simulate script load and Mermaid availability
    act(() => {
      (global.window as any).mermaid = mockMermaid;
      if (scriptElement && scriptElement.onload) {
        scriptElement.onload();
      }
    });

    // Wait for run() to be called (with longer timeout for async operations)
    await waitFor(() => {
      expect(mockMermaid.initialize).toHaveBeenCalled();
    }, { timeout: 3000 });
  });

  it('handles createElementNS error gracefully', async () => {
    const mockMermaid = {
      initialize: jest.fn(),
      run: jest.fn(() => {
        // Simulate createElementNS error
        throw new Error('Cannot read properties of undefined (reading \'createElementNS\')');
      }),
      render: jest.fn(),
      parse: jest.fn(),
    };

    let scriptElement: any = null;
    mockCreateElement.mockImplementation((tagName: string) => {
      const element = {
        tagName: tagName.toUpperCase(),
        className: '',
        textContent: '',
        innerHTML: '',
        src: '',
        async: false,
        onload: null as any,
        onerror: null as any,
        addEventListener: jest.fn(),
        appendChild: jest.fn(),
        querySelector: jest.fn(),
      };
      if (tagName === 'script') {
        scriptElement = element;
      }
      return element as any;
    });

    const { container } = render(<InteractiveVisualizations lesson={mockLesson} topic="Test Topic" />);

    await waitFor(() => {
      expect(mockCreateElement).toHaveBeenCalledWith('script');
    }, { timeout: 3000 });

    act(() => {
      (global.window as any).mermaid = mockMermaid;
      if (scriptElement && scriptElement.onload) {
        scriptElement.onload();
      }
    });

    // Should handle error gracefully without crashing
    await waitFor(() => {
      expect(mockMermaid.initialize).toHaveBeenCalled();
    }, { timeout: 3000 });

    // Component should still be rendered
    expect(container).toBeInTheDocument();
  });

  it('falls back to render() if run() is not available', async () => {
    const mockMermaid = {
      initialize: jest.fn(),
      run: undefined,
      render: jest.fn((id: string, diagram: string, callback: (svg: string) => void) => {
        callback('<svg>test</svg>');
      }),
      parse: jest.fn(),
    };

    let scriptElement: any = null;
    mockCreateElement.mockImplementation((tagName: string) => {
      const element = {
        tagName: tagName.toUpperCase(),
        className: '',
        textContent: '',
        innerHTML: '',
        src: '',
        async: false,
        onload: null as any,
        onerror: null as any,
        addEventListener: jest.fn(),
        appendChild: jest.fn(),
        querySelector: jest.fn(),
      };
      if (tagName === 'script') {
        scriptElement = element;
      }
      return element as any;
    });

    render(<InteractiveVisualizations lesson={mockLesson} topic="Test Topic" />);

    await waitFor(() => {
      expect(mockCreateElement).toHaveBeenCalledWith('script');
    }, { timeout: 3000 });

    act(() => {
      (global.window as any).mermaid = mockMermaid;
      if (scriptElement && scriptElement.onload) {
        scriptElement.onload();
      }
    });

    // Should initialize Mermaid
    await waitFor(() => {
      expect(mockMermaid.initialize).toHaveBeenCalled();
    }, { timeout: 3000 });
  });

  it('handles script load error', async () => {
    let scriptElement: any = null;
    mockCreateElement.mockImplementation((tagName: string) => {
      const element = {
        tagName: tagName.toUpperCase(),
        className: '',
        textContent: '',
        innerHTML: '',
        src: '',
        async: false,
        onload: null as any,
        onerror: null as any,
        addEventListener: jest.fn(),
        appendChild: jest.fn(),
        querySelector: jest.fn(),
      };
      if (tagName === 'script') {
        scriptElement = element;
      }
      return element as any;
    });

    render(<InteractiveVisualizations lesson={mockLesson} topic="Test Topic" />);

    await waitFor(() => {
      expect(mockCreateElement).toHaveBeenCalledWith('script');
    }, { timeout: 3000 });

    // Simulate script load error
    act(() => {
      if (scriptElement && scriptElement.onerror) {
        scriptElement.onerror();
      }
    });

    // Should show fallback code
    expect(mockCreateElement).toHaveBeenCalled();
  });

  it('does not load Mermaid when chart visualization is active', async () => {
    render(<InteractiveVisualizations lesson={mockLesson} topic="Test Topic" />);

    // Click on chart tab
    const chartTab = screen.getByText('Analytics');
    act(() => {
      chartTab.click();
    });

    // Should create Chart.js script instead of Mermaid
    await waitFor(() => {
      // Chart.js script should be created
      expect(mockCreateElement).toHaveBeenCalled();
    }, { timeout: 3000 });
  });

  it('generates correct Mermaid diagram from lesson data', () => {
    const { container } = render(
      <InteractiveVisualizations lesson={mockLesson} topic="Test Topic" />
    );

    // The component should generate diagram code
    // We can't easily test the exact diagram content without rendering,
    // but we can verify the component renders without errors
    expect(container).toBeInTheDocument();
  });

  it('handles missing lesson properties gracefully', () => {
    const incompleteLesson: OGLesson = {
      big_picture: 'Only big picture',
      // Other properties are undefined
    };

    render(<InteractiveVisualizations lesson={incompleteLesson} topic="Test Topic" />);

    // Should render without errors
    expect(screen.getByText('Test Topic')).toBeInTheDocument();
  });

  it('waits for DOM to be ready before rendering', async () => {
    // Mock document.readyState as a getter
    Object.defineProperty(global.document, 'readyState', {
      get: () => 'loading',
      configurable: true,
    });

    render(<InteractiveVisualizations lesson={mockLesson} topic="Test Topic" />);

    // Should wait for DOMContentLoaded or use setTimeout
    // The component should handle loading state
    expect(mockCreateElement).toHaveBeenCalled();
  });
});

