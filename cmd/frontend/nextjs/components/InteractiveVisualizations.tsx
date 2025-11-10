import React, { useState, useEffect, useRef } from 'react';
import { OGLesson } from '../types';

interface InteractiveVisualizationsProps {
  lesson: OGLesson;
  topic: string;
}

type VisualizationType = 'mermaid' | 'chart' | 'flowchart' | 'mindmap' | 'observable';

const InteractiveVisualizations: React.FC<InteractiveVisualizationsProps> = ({ lesson, topic }) => {
  const [activeViz, setActiveViz] = useState<VisualizationType>('mermaid');
  const mermaidRef = useRef<HTMLDivElement>(null);
  
  // Generate Mermaid diagram from lesson content
  const generateMermaidDiagram = (): string => {
    // Escape special characters that might break Mermaid syntax
    const escapeMermaidText = (text: string | undefined, maxLength: number = 60): string => {
      if (!text) return '';
      // Clean and truncate text
      let cleaned = text
        .replace(/"/g, '&quot;')
        .replace(/'/g, '&#39;')
        .replace(/\n/g, ' ')
        .replace(/\r/g, '')
        .replace(/\s+/g, ' ')
        .trim();
      
      if (cleaned.length > maxLength) {
        cleaned = cleaned.substring(0, maxLength) + '...';
      }
      return cleaned;
    };

    // Extract key phrases from each section for better visualization
    const extractKeyPhrase = (text: string | undefined, maxWords: number = 8): string => {
      if (!text) return '';
      const words = text.split(/\s+/).filter(w => w.length > 0);
      if (words.length <= maxWords) return escapeMermaidText(text, 80);
      return escapeMermaidText(words.slice(0, maxWords).join(' '), 80) + '...';
    };

    const escapedTopic = escapeMermaidText(topic, 50);
    const bigPicture = extractKeyPhrase(lesson.big_picture) || 'Overview';
    const metaphor = extractKeyPhrase(lesson.metaphor) || 'Analogy';
    const mechanism = extractKeyPhrase(lesson.core_mechanism) || 'How it works';
    const examples = extractKeyPhrase(lesson.toy_example_code) || 'Code examples';
    const applications = extractKeyPhrase(lesson.real_life) || 'Real world applications';
    const bestPractices = extractKeyPhrase(lesson.best_practices) || 'Best practices';
    const memoryHook = extractKeyPhrase(lesson.memory_hook) || 'Memory aid';

    return `graph TD
    A["${escapedTopic}"]
    B["Big Picture<br/>${bigPicture}"]
    C["Metaphor<br/>${metaphor}"]
    D["Core Mechanism<br/>${mechanism}"]
    E["Examples<br/>${examples}"]
    F["Applications<br/>${applications}"]
    G["Best Practices<br/>${bestPractices}"]
    H["Memory Hook<br/>${memoryHook}"]
    
    A --> B
    A --> C
    B --> D
    C --> D
    D --> E
    E --> F
    F --> G
    D --> H
    H --> G
    
    style A fill:#3b82f6,stroke:#1e40af,stroke-width:3px,color:#fff
    style B fill:#10b981,stroke:#059669,stroke-width:2px,color:#fff
    style C fill:#8b5cf6,stroke:#6d28d9,stroke-width:2px,color:#fff
    style D fill:#f59e0b,stroke:#d97706,stroke-width:2px,color:#fff
    style E fill:#ef4444,stroke:#dc2626,stroke-width:2px,color:#fff
    style F fill:#06b6d4,stroke:#0891b2,stroke-width:2px,color:#fff
    style G fill:#ec4899,stroke:#db2777,stroke-width:2px,color:#fff
    style H fill:#f97316,stroke:#ea580c,stroke-width:2px,color:#fff`;
  };

  // Generate flowchart diagram from lesson content
  const generateFlowchart = (): string => {
    // Escape text for flowchart labels
    const escapeFlowchartText = (text: string | undefined, maxLength: number = 40): string => {
      if (!text) return '';
      let cleaned = text
        .replace(/"/g, '&quot;')
        .replace(/'/g, '&#39;')
        .replace(/\n/g, ' ')
        .replace(/\r/g, '')
        .replace(/\s+/g, ' ')
        .trim();
      
      if (cleaned.length > maxLength) {
        cleaned = cleaned.substring(0, maxLength) + '...';
      }
      return cleaned;
    };

    // Extract meaningful snippets from each section
    const getSectionSnippet = (text: string | undefined, label: string): string => {
      if (!text || text.trim().length === 0) return label;
      const snippet = escapeFlowchartText(text, 35);
      return snippet.length > 0 ? snippet : label;
    };

    const bigPictureSnippet = getSectionSnippet(lesson.big_picture, 'Big Picture');
    const metaphorSnippet = getSectionSnippet(lesson.metaphor, 'Metaphor');
    const mechanismSnippet = getSectionSnippet(lesson.core_mechanism, 'Mechanism');
    const examplesSnippet = getSectionSnippet(lesson.toy_example_code, 'Examples');
    const applicationsSnippet = getSectionSnippet(lesson.real_life, 'Applications');
    const bestPracticesSnippet = getSectionSnippet(lesson.best_practices, 'Best Practices');
    const escapedTopic = escapeFlowchartText(topic, 30);

    return `flowchart LR
    Start([Start: ${escapedTopic}]) --> Understand["${bigPictureSnippet}"]
    Understand --> Relate["${metaphorSnippet}"]
    Relate --> Learn["${mechanismSnippet}"]
    Learn --> Practice["${examplesSnippet}"]
    Practice --> Apply["${applicationsSnippet}"]
    Apply --> Master["${bestPracticesSnippet}"]
    Master --> End([Complete])
    
    style Start fill:#3b82f6,stroke:#1e40af,color:#fff
    style End fill:#10b981,stroke:#059669,color:#fff
    style Understand fill:#fbbf24,stroke:#f59e0b,color:#000
    style Relate fill:#a78bfa,stroke:#8b5cf6,color:#fff
    style Learn fill:#fb7185,stroke:#f43f5e,color:#fff
    style Practice fill:#34d399,stroke:#10b981,color:#000
    style Apply fill:#60a5fa,stroke:#3b82f6,color:#fff
    style Master fill:#f472b6,stroke:#ec4899,color:#fff`;
  };

  // Generate mind map from lesson content
  const generateMindMap = (): string => {
    // Escape special characters for mind map
    const escapeText = (text: string | undefined, maxLength: number = 60): string => {
      if (!text) return '';
      let cleaned = text
        .replace(/"/g, '&quot;')
        .replace(/'/g, '&#39;')
        .replace(/\n/g, ' ')
        .replace(/\r/g, '')
        .replace(/\s+/g, ' ')
        .trim();
      
      if (cleaned.length > maxLength) {
        // Try to break at word boundary
        const truncated = cleaned.substring(0, maxLength);
        const lastSpace = truncated.lastIndexOf(' ');
        cleaned = lastSpace > 0 ? truncated.substring(0, lastSpace) + '...' : truncated + '...';
      }
      return cleaned;
    };

    // Extract key concepts from each section
    const extractKeyConcept = (text: string | undefined, defaultLabel: string): string => {
      if (!text || text.trim().length === 0) return defaultLabel;
      // Get first sentence or first 60 chars
      const firstSentence = text.split(/[.!?]/)[0].trim();
      return escapeText(firstSentence || text, 60) || defaultLabel;
    };

    const bigPicture = extractKeyConcept(lesson.big_picture, 'Overview');
    const metaphor = extractKeyConcept(lesson.metaphor, 'Analogy');
    const mechanism = extractKeyConcept(lesson.core_mechanism, 'How it works');
    const examples = extractKeyConcept(lesson.toy_example_code, 'Code examples');
    const applications = extractKeyConcept(lesson.real_life, 'Real world');
    const bestPractices = extractKeyConcept(lesson.best_practices, 'Best practices');
    const memoryHook = extractKeyConcept(lesson.memory_hook, 'Memory aid');

    const escapedTopic = escapeText(topic, 40);

    return `mindmap
  root((${escapedTopic}))
    Big Picture
      ${bigPicture}
    Metaphor
      ${metaphor}
    Core Mechanism
      ${mechanism}
    Examples
      ${examples}
    Applications
      ${applications}
    Best Practices
      ${bestPractices}
    Memory Hook
      ${memoryHook}`;
  };

  // Load Mermaid dynamically (CDN only to avoid ESM issues)
  useEffect(() => {
    // Only run in browser
    if (typeof window === 'undefined' || typeof document === 'undefined') {
      return;
    }

    if (activeViz === 'mermaid' || activeViz === 'flowchart' || activeViz === 'mindmap') {
      // Wait for DOM to be fully ready
      const waitForDOM = () => {
        if (document.readyState === 'loading') {
          document.addEventListener('DOMContentLoaded', () => {
            setTimeout(loadMermaid, 100);
          });
          return;
        }
        // DOM is already ready
        setTimeout(loadMermaid, 100);
      };

      const loadMermaid = () => {
        // Double-check we're in browser with DOM ready
        if (typeof window === 'undefined' || typeof document === 'undefined' || !document.body) {
          console.warn('DOM not ready for Mermaid');
          showDiagramCode();
          return;
        }

        // Check if Mermaid is already loaded and initialized
        if ((window as any).mermaid && typeof (window as any).mermaid.initialize === 'function') {
          // Ensure DOM element exists
          if (mermaidRef.current) {
            console.log('Mermaid already loaded, rendering immediately');
            setTimeout(() => {
              renderMermaid().catch(err => {
                console.error('Failed to render Mermaid:', err);
                showDiagramCode();
              });
            }, 300);
          }
          return;
        }

        // Check if script is already being loaded
        const existingScript = document.querySelector('script[src*="mermaid"]');
        if (existingScript) {
          existingScript.addEventListener('load', () => {
            setTimeout(() => {
              initializeAndRender();
            }, 300);
          });
          return;
        }

        const script = document.createElement('script');
        // Use Mermaid v9 which is more stable and compatible
        script.src = 'https://cdn.jsdelivr.net/npm/mermaid@9/dist/mermaid.min.js';
        script.async = true;
        script.onload = () => {
          console.log('Mermaid script loaded');
          // Wait for Mermaid to be fully available and DOM to be ready
          setTimeout(() => {
            initializeAndRender();
          }, 500);
        };
        script.onerror = () => {
          console.error('Failed to load Mermaid script');
          showDiagramCode();
        };
        document.head.appendChild(script);
      };

      const initializeAndRender = () => {
        // Ensure we're in browser with DOM ready
        if (typeof window === 'undefined' || typeof document === 'undefined' || !document.body) {
          console.warn('DOM not ready for Mermaid initialization');
          showDiagramCode();
          return;
        }

        const mermaid = (window as any).mermaid;
        if (!mermaid || typeof mermaid.initialize !== 'function') {
          console.error('Mermaid not properly loaded');
          showDiagramCode();
          return;
        }

        try {
          // Initialize Mermaid with proper config
          // Note: startOnLoad should be true for run() to work properly
          mermaid.initialize({ 
            startOnLoad: true, // Allow auto-start for run() method
            theme: 'default',
            securityLevel: 'loose',
            flowchart: {
              useMaxWidth: true,
              htmlLabels: true,
              curve: 'basis'
            }
          });

          // Wait a bit more for initialization to complete
          setTimeout(() => {
            renderMermaid().catch(err => {
              console.error('Failed to render Mermaid:', err);
              showDiagramCode();
            });
          }, 500);
        } catch (initError) {
          console.error('Failed to initialize Mermaid:', initError);
          showDiagramCode();
        }
      };

      const renderMermaid = async () => {
        // Ensure DOM is ready and Mermaid is available
        if (typeof window === 'undefined' || typeof document === 'undefined' || !document.body) {
          console.warn('DOM not ready for Mermaid rendering');
          return;
        }

        if (!mermaidRef.current) {
          console.warn('Mermaid ref not available');
          return;
        }

        const mermaid = (window as any).mermaid;
        if (!mermaid) {
          console.error('Mermaid not available');
          showDiagramCode();
          return;
        }

        const diagram = activeViz === 'mermaid' 
          ? generateMermaidDiagram()
          : activeViz === 'flowchart'
          ? generateFlowchart()
          : generateMindMap();
        
        console.log('Rendering Mermaid diagram:', { activeViz, diagramLength: diagram.length });
        
        try {
          // Clear previous content
          if (mermaidRef.current) {
            mermaidRef.current.innerHTML = '<div class="text-gray-500 text-center py-4">Rendering diagram...</div>';
          }
          
          // Use the safer run() method with pre tag instead of renderAsync
          // This avoids the createElementNS issue
          if (mermaidRef.current) {
            // Create a pre element with mermaid class
            const preElement = document.createElement('pre');
            preElement.className = 'mermaid';
            preElement.textContent = diagram;
            preElement.style.display = 'block';
            preElement.style.textAlign = 'center';
            
            // Clear and append
            mermaidRef.current.innerHTML = '';
            mermaidRef.current.appendChild(preElement);
            
            // Wait for DOM to update, then run Mermaid
            setTimeout(() => {
              try {
                // Use run() method which is more compatible - it processes all .mermaid elements
                if (mermaid && typeof mermaid.run === 'function') {
                  console.log('Using mermaid.run() method');
                  // run() without parameters processes all elements with class 'mermaid'
                  const result = mermaid.run();
                  console.log('Mermaid run result:', result);
                  
                  // Check if rendering was successful after a short delay
                  setTimeout(() => {
                    if (mermaidRef.current) {
                      const svg = mermaidRef.current.querySelector('svg');
                      if (!svg) {
                        console.warn('No SVG found after mermaid.run(), trying fallback');
                        // Try alternative rendering
                        tryAlternativeRender();
                      } else {
                        console.log('Mermaid diagram rendered successfully');
                        // Center the SVG
                        svg.style.display = 'block';
                        svg.style.margin = '0 auto';
                        svg.style.maxWidth = '100%';
                        svg.style.height = 'auto';
                      }
                    }
                  }, 1000);
                } else {
                  tryAlternativeRender();
                }
              } catch (runError) {
                console.error('Mermaid run error:', runError);
                // If run fails, try alternative rendering
                tryAlternativeRender();
              }
            }, 300);
          }
        } catch (error) {
          console.error('Mermaid rendering error:', error);
          showDiagramCode();
        }

        const tryAlternativeRender = () => {
          if (!mermaidRef.current || !mermaid) return;
          
          const diagram = activeViz === 'mermaid' 
            ? generateMermaidDiagram()
            : activeViz === 'flowchart'
            ? generateFlowchart()
            : generateMindMap();
          
          try {
            if (mermaid && typeof mermaid.render === 'function') {
              console.log('Trying mermaid.render() as fallback');
              const id = `mermaid-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
              mermaid.render(id, diagram, (svgCode: string) => {
                if (mermaidRef.current && svgCode) {
                  mermaidRef.current.innerHTML = svgCode;
                  const svg = mermaidRef.current.querySelector('svg');
                  if (svg) {
                    svg.style.display = 'block';
                    svg.style.margin = '0 auto';
                    svg.style.maxWidth = '100%';
                    svg.style.height = 'auto';
                  }
                  console.log('Mermaid diagram rendered via render()');
                }
              });
            } else {
              showDiagramCode();
            }
          } catch (renderError) {
            console.error('Alternative render error:', renderError);
            showDiagramCode();
          }
        };
      };

      const showDiagramCode = () => {
        if (mermaidRef.current) {
          const diagram = activeViz === 'mermaid' 
            ? generateMermaidDiagram()
            : activeViz === 'flowchart'
            ? generateFlowchart()
            : generateMindMap();
          mermaidRef.current.innerHTML = `
            <div class="bg-blue-50 border border-blue-200 rounded-lg p-4 mb-4">
              <p class="text-sm text-blue-800 mb-2">
                <strong>Note:</strong> Diagram code (Mermaid rendering unavailable)
              </p>
            </div>
            <pre class="bg-gray-100 p-4 rounded overflow-auto text-xs"><code>${diagram}</code></pre>
          `;
        }
      };
      
      waitForDOM();
    }
  }, [activeViz, lesson]);

  // Chart data for Chart.js
  const chartData = {
    labels: ['Big Picture', 'Metaphor', 'Core Mechanism', 'Examples', 'Real Life', 'Best Practices'],
    datasets: [{
      label: 'Content Length',
      data: [
        lesson.big_picture?.length || 0,
        lesson.metaphor?.length || 0,
        lesson.core_mechanism?.length || 0,
        lesson.toy_example_code?.length || 0,
        lesson.real_life?.length || 0,
        lesson.best_practices?.length || 0,
      ],
      backgroundColor: [
        'rgba(59, 130, 246, 0.6)',
        'rgba(16, 185, 129, 0.6)',
        'rgba(139, 92, 246, 0.6)',
        'rgba(245, 158, 11, 0.6)',
        'rgba(6, 182, 212, 0.6)',
        'rgba(236, 72, 153, 0.6)',
      ],
      borderColor: [
        'rgba(59, 130, 246, 1)',
        'rgba(16, 185, 129, 1)',
        'rgba(139, 92, 246, 1)',
        'rgba(245, 158, 11, 1)',
        'rgba(6, 182, 212, 1)',
        'rgba(236, 72, 153, 1)',
      ],
      borderWidth: 2,
    }],
  };

  // Load Chart.js dynamically (CDN only to avoid ESM issues)
  useEffect(() => {
    if (activeViz === 'chart') {
      const loadChart = () => {
        if (typeof window === 'undefined') return;
        
        if ((window as any).Chart) {
          // Small delay to ensure DOM is ready
          setTimeout(() => renderChart(), 100);
          return;
        }

        const script = document.createElement('script');
        script.src = 'https://cdn.jsdelivr.net/npm/chart.js@4/dist/chart.umd.min.js';
        script.onload = () => {
          if ((window as any).Chart) {
            // Small delay to ensure DOM is ready
            setTimeout(() => renderChart(), 100);
          }
        };
        script.onerror = () => {
          showChartFallback();
        };
        document.head.appendChild(script);
      };

      const renderChart = () => {
        const ctx = document.getElementById('lessonChart') as HTMLCanvasElement;
        if (ctx && (window as any).Chart) {
          // Destroy existing chart if it exists
          const existingChart = (ctx as any).chart;
          if (existingChart) {
            existingChart.destroy();
          }
          
          const Chart = (window as any).Chart;
          const newChart = new Chart(ctx, {
            type: 'bar',
            data: chartData,
            options: {
              responsive: true,
              maintainAspectRatio: false,
              plugins: {
                legend: {
                  display: false,
                },
                title: {
                  display: true,
                  text: 'Lesson Content Distribution',
                  font: {
                    size: 16,
                  },
                },
              },
              scales: {
                y: {
                  beginAtZero: true,
                },
              },
            },
          });
          
          // Store chart instance for cleanup
          (ctx as any).chart = newChart;
        }
      };

      const showChartFallback = () => {
        const container = document.getElementById('chartContainer');
        if (container) {
          container.innerHTML = `
            <div class="bg-blue-50 border border-blue-200 rounded-lg p-4 mb-4">
              <p class="text-sm text-blue-800 mb-2">
                <strong>Note:</strong> Loading Chart.js from CDN...
              </p>
            </div>
            <div class="space-y-2">
              ${chartData.labels.map((label, i) => `
                <div class="flex items-center space-x-4">
                  <span class="w-32 text-sm font-medium">${label}</span>
                  <div class="flex-1 bg-gray-200 rounded-full h-6">
                    <div class="bg-blue-500 h-6 rounded-full" style="width: ${(chartData.datasets[0].data[i] / Math.max(...chartData.datasets[0].data)) * 100}%"></div>
                  </div>
                  <span class="text-sm text-gray-600 w-16">${chartData.datasets[0].data[i]}</span>
                </div>
              `).join('')}
            </div>
          `;
        }
      };
      
      loadChart();
      
      // Cleanup function
      return () => {
        const ctx = document.getElementById('lessonChart') as HTMLCanvasElement;
        if (ctx && (ctx as any).chart) {
          (ctx as any).chart.destroy();
        }
      };
    }
  }, [activeViz, chartData]);

  const visualizationTabs = [
    { id: 'mermaid' as VisualizationType, label: 'Concept Map', icon: '🗺️' },
    { id: 'flowchart' as VisualizationType, label: 'Learning Flow', icon: '📊' },
    { id: 'mindmap' as VisualizationType, label: 'Mind Map', icon: '🧠' },
    { id: 'chart' as VisualizationType, label: 'Analytics', icon: '📈' },
  ];

  return (
    <div className="space-y-6 animate-fade-in">
      {/* Header */}
      <div className="bg-gradient-to-r from-purple-500 via-indigo-600 to-purple-600 text-white p-6 rounded-lg shadow-lg">
        <h2 className="text-2xl font-bold mb-2">{topic}</h2>
        <p className="text-purple-100 font-medium">Interactive Visualizations</p>
      </div>

      {/* Visualization Tabs */}
      <div className="bg-white rounded-lg shadow-md p-4">
        <div className="flex flex-wrap gap-2 border-b border-gray-200 pb-2">
          {visualizationTabs.map((tab) => (
            <button
              key={tab.id}
              onClick={() => setActiveViz(tab.id)}
              className={`px-4 py-2 rounded-lg font-medium transition-all duration-200 ${
                activeViz === tab.id
                  ? 'bg-gradient-to-r from-purple-500 to-indigo-500 text-white shadow-md'
                  : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
              }`}
            >
              <span className="mr-2">{tab.icon}</span>
              {tab.label}
            </button>
          ))}
        </div>

        {/* Visualization Content */}
        <div className="mt-6 min-h-[500px]">
          {activeViz === 'mermaid' || activeViz === 'flowchart' || activeViz === 'mindmap' ? (
            <div 
              ref={mermaidRef} 
              className="w-full bg-white rounded-lg p-6 border border-gray-200 overflow-auto"
              style={{ minHeight: '500px' }}
            >
              <div className="text-gray-500 text-center py-8">Loading diagram...</div>
            </div>
          ) : activeViz === 'chart' ? (
            <div id="chartContainer" className="h-[500px] bg-white rounded-lg p-4">
              <canvas id="lessonChart"></canvas>
            </div>
          ) : null}
        </div>
      </div>

      {/* Embed External Visualization Option */}
      <div className="bg-white rounded-lg shadow-md p-6">
        <h3 className="text-lg font-semibold text-gray-900 mb-4 flex items-center space-x-2">
          <span>🔗</span>
          <span>Embed External Visualization</span>
        </h3>
        <div className="space-y-4">
          <div className="bg-gray-50 rounded-lg p-4">
            <p className="text-sm text-gray-600 mb-2">
              You can embed visualizations from external tools like:
            </p>
            <ul className="list-disc list-inside text-sm text-gray-700 space-y-1">
              <li><strong>Observable:</strong> Embed interactive notebooks and visualizations</li>
              <li><strong>Plotly:</strong> Embed interactive charts and graphs</li>
              <li><strong>Tableau:</strong> Embed business intelligence dashboards</li>
              <li><strong>Google Charts:</strong> Embed Google visualization tools</li>
            </ul>
          </div>
          <div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
            <p className="text-sm text-blue-800">
              <strong>Tip:</strong> To embed an external visualization, add an iframe or embed code in the lesson content.
            </p>
          </div>
        </div>
      </div>
    </div>
  );
};

export default InteractiveVisualizations;


