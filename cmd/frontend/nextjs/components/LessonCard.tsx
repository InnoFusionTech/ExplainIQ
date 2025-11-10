import React, { useState } from 'react';
import { OGLesson, ImageRef, PDFResponse } from '../types';
import InteractiveVisualizations from './InteractiveVisualizations';
import { getOrchestratorURL } from '../utils/getOrchestratorURL';

interface LessonCardProps {
  lesson: OGLesson;
  images?: ImageRef[];
  sessionId?: string;
  explanationType?: string;
  topic?: string;
  onSave?: () => void;
}

const SECTION_ICONS: Record<string, string> = {
  big_picture: '🌍',
  metaphor: '💭',
  core_mechanism: '⚙️',
  toy_example_code: '💻',
  memory_hook: '🎣',
  real_life: '🌐',
  best_practices: '⭐',
};

const LessonCard: React.FC<LessonCardProps> = ({ lesson, images = [], sessionId, explanationType = 'standard', topic, onSave }) => {
  const [isGeneratingPDF, setIsGeneratingPDF] = useState(false);
  const [pdfError, setPdfError] = useState<string | null>(null);
  const [expandedSections, setExpandedSections] = useState<Set<string>>(new Set(['big_picture', 'core_mechanism']));
  const [isSaving, setIsSaving] = useState(false);
  const [saveSuccess, setSaveSuccess] = useState(false);
  const [saveError, setSaveError] = useState<string | null>(null);

  const handleDownloadPDF = async () => {
    if (!sessionId) {
      setPdfError('Session ID is required for PDF generation');
      return;
    }

    setIsGeneratingPDF(true);
    setPdfError(null);

    try {
      const response = await fetch(`/api/sessions/${sessionId}/pdf`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
      });

      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.error || 'Failed to generate PDF');
      }

      const pdfData: PDFResponse = await response.json();
      
      // Create a temporary link to download the PDF
      const link = document.createElement('a');
      link.href = pdfData.pdf_url;
      link.download = pdfData.filename;
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
    } catch (error) {
      setPdfError(error instanceof Error ? error.message : 'PDF generation failed');
    } finally {
      setIsGeneratingPDF(false);
    }
  };

  const handleSave = async () => {
    if (!sessionId) {
      setSaveError('Session ID is required');
      return;
    }

    setIsSaving(true);
    setSaveError(null);
    setSaveSuccess(false);

    try {
      const orchestratorURL = getOrchestratorURL();
      
      const requestBody = {
        session_id: sessionId,
        user_id: sessionId, // Use session ID as user ID for now
        title: topic || lesson.big_picture?.substring(0, 50) || 'Saved Lesson',
        explanation_type: explanationType,
        result: {
          lesson: JSON.stringify(lesson), // Backend expects lesson as JSON string
          images: images?.reduce((acc, img, idx) => {
            acc[`image_${idx}`] = img.url;
            return acc;
          }, {} as Record<string, string>) || {},
          summary: lesson.big_picture || '',
        },
      };

      console.log('Saving lesson with request:', JSON.stringify(requestBody, null, 2));

      const response = await fetch(`${orchestratorURL}/api/saved`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(requestBody),
      });

      if (!response.ok) {
        let errorMessage = 'Failed to save lesson';
        const contentType = response.headers.get('content-type');
        if (contentType && contentType.includes('application/json')) {
          try {
            const errorData = await response.json();
            errorMessage = errorData.error || errorData.message || errorMessage;
          } catch (parseError) {
            // Fallback if JSON parsing fails
            errorMessage = 'Failed to save lesson';
          }
        } else {
          // If not JSON, try to get text
          try {
            const text = await response.text();
            errorMessage = text || errorMessage;
          } catch (textError) {
            // Use default error message
            errorMessage = `Failed to save lesson (${response.status})`;
          }
        }
        throw new Error(errorMessage);
      }

      setSaveSuccess(true);
      setTimeout(() => setSaveSuccess(false), 3000);
      if (onSave) {
        onSave();
      }
    } catch (error) {
      setSaveError(error instanceof Error ? error.message : 'Failed to save lesson');
    } finally {
      setIsSaving(false);
    }
  };

  const sections = [
    { key: 'big_picture', title: 'Big Picture', content: lesson.big_picture, color: 'border-blue-500' },
    { key: 'metaphor', title: 'Metaphor', content: lesson.metaphor, color: 'border-green-500' },
    { key: 'core_mechanism', title: 'Core Mechanism', content: lesson.core_mechanism, color: 'border-purple-500' },
    { key: 'toy_example_code', title: 'Toy Example', content: lesson.toy_example_code, color: 'border-orange-500', isCode: true },
    { key: 'memory_hook', title: 'Memory Hook', content: lesson.memory_hook, color: 'border-pink-500' },
    { key: 'real_life', title: 'Real Life', content: lesson.real_life, color: 'border-indigo-500' },
    { key: 'best_practices', title: 'Best Practices', content: lesson.best_practices, color: 'border-yellow-500' },
  ];

  return (
    <div className="space-y-6 animate-fade-in">
      {/* Interactive Visualizations */}
      <div className="mb-6">
        <InteractiveVisualizations lesson={lesson} topic={lesson.big_picture?.substring(0, 50) || 'Lesson'} />
      </div>

      {/* Action Buttons */}
      {sessionId && (
        <div className="flex justify-end gap-3 mb-6">
          <button
            onClick={handleSave}
            disabled={isSaving}
            className={`px-6 py-3 rounded-lg font-semibold focus:ring-2 focus:ring-offset-2 disabled:opacity-50 disabled:cursor-not-allowed transition-all duration-200 shadow-md hover:shadow-lg transform hover:-translate-y-0.5 flex items-center space-x-2 ${
              saveSuccess
                ? 'bg-green-600 text-white hover:bg-green-700 focus:ring-green-500'
                : 'bg-gradient-to-r from-blue-600 to-indigo-600 text-white hover:from-blue-700 hover:to-indigo-700 focus:ring-blue-500'
            }`}
          >
            {isSaving ? (
              <>
                <svg className="animate-spin h-4 w-4" fill="none" viewBox="0 0 24 24">
                  <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                  <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                </svg>
                <span>Saving...</span>
              </>
            ) : saveSuccess ? (
              <>
                <svg className="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                </svg>
                <span>Saved!</span>
              </>
            ) : (
              <>
                <svg className="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 5a2 2 0 012-2h10a2 2 0 012 2v16l-7-3.5L5 21V5z" />
                </svg>
                <span>Save for Later</span>
              </>
            )}
          </button>
          <button
            onClick={handleDownloadPDF}
            disabled={isGeneratingPDF}
            className="bg-gradient-to-r from-red-600 to-pink-600 text-white px-6 py-3 rounded-lg font-semibold hover:from-red-700 hover:to-pink-700 focus:ring-2 focus:ring-red-500 focus:ring-offset-2 disabled:opacity-50 disabled:cursor-not-allowed transition-all duration-200 shadow-md hover:shadow-lg transform hover:-translate-y-0.5 flex items-center space-x-2"
          >
            {isGeneratingPDF ? (
              <>
                <svg className="animate-spin h-4 w-4" fill="none" viewBox="0 0 24 24">
                  <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                  <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                </svg>
                <span>Generating PDF...</span>
              </>
            ) : (
              <>
                <svg className="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 10v6m0 0l-3-3m3 3l3-3m2 8H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
                </svg>
                <span>Download PDF</span>
              </>
            )}
          </button>
        </div>
      )}

      {/* Save Success/Error Display */}
      {saveSuccess && (
        <div className="bg-green-50 border border-green-200 rounded-lg p-4 mb-6 animate-fade-in">
          <div className="flex items-center">
            <svg className="h-5 w-5 text-green-400 mr-3" fill="currentColor" viewBox="0 0 20 20">
              <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clipRule="evenodd" />
            </svg>
            <p className="text-sm text-green-800 font-medium">Lesson saved successfully!</p>
          </div>
        </div>
      )}
      {saveError && (
        <div className="bg-red-50 border border-red-200 rounded-lg p-4 mb-6 animate-fade-in">
          <div className="flex">
            <div className="flex-shrink-0">
              <svg className="h-5 w-5 text-red-400" viewBox="0 0 20 20" fill="currentColor">
                <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clipRule="evenodd" />
              </svg>
            </div>
            <div className="ml-3">
              <h3 className="text-sm font-medium text-red-800">Save Error</h3>
              <p className="text-sm text-red-700 mt-1">{saveError}</p>
            </div>
          </div>
        </div>
      )}

      {/* PDF Error Display */}
      {pdfError && (
        <div className="bg-red-50 border border-red-200 rounded-lg p-4 mb-6">
          <div className="flex">
            <div className="flex-shrink-0">
              <svg className="h-5 w-5 text-red-400" viewBox="0 0 20 20" fill="currentColor">
                <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clipRule="evenodd" />
              </svg>
            </div>
            <div className="ml-3">
              <h3 className="text-sm font-medium text-red-800">PDF Generation Error</h3>
              <p className="text-sm text-red-700 mt-1">{pdfError}</p>
            </div>
          </div>
        </div>
      )}

      {sections.map((section) => {
        const isExpanded = expandedSections.has(section.key);
        const isCollapsible = section.key === 'toy_example_code' || section.key === 'real_life';
        const icon = SECTION_ICONS[section.key] || '📋';

        return (
          <div 
            key={section.key} 
            className={`border-l-4 pl-6 py-4 bg-white rounded-r-lg shadow-md hover:shadow-lg transition-all duration-200 ${section.color}`}
          >
            <button
              onClick={() => {
                if (isCollapsible) {
                  setExpandedSections(prev => {
                    const next = new Set(prev);
                    if (isExpanded) {
                      next.delete(section.key);
                    } else {
                      next.add(section.key);
                    }
                    return next;
                  });
                }
              }}
              className={`w-full flex items-center justify-between mb-3 ${isCollapsible ? 'cursor-pointer' : ''}`}
            >
              <h3 className="text-lg font-bold text-gray-900 flex items-center space-x-2">
                <span className="text-xl">{icon}</span>
                <span>{section.title}</span>
              </h3>
              {isCollapsible && (
                <svg
                  className={`w-5 h-5 text-gray-500 transition-transform duration-300 ${isExpanded ? 'transform rotate-180' : ''}`}
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
                </svg>
              )}
            </button>
            <div
              className={`overflow-hidden transition-all duration-300 ease-in-out ${
                isCollapsible && !isExpanded ? 'max-h-0 opacity-0' : 'max-h-[5000px] opacity-100'
              }`}
            >
              {section.isCode ? (
                <pre className="bg-gray-900 text-gray-100 p-4 rounded-lg text-sm overflow-x-auto border border-gray-700 shadow-inner">
                  <code className="font-mono">{section.content}</code>
                </pre>
              ) : (
                <p className="text-gray-700 leading-relaxed whitespace-pre-wrap">{section.content}</p>
              )}
            </div>
          </div>
        );
      })}

      {/* Visualizations */}
      {images && images.length > 0 && (
        <div className="bg-gradient-to-br from-purple-50 to-indigo-50 rounded-lg p-6 border border-purple-200">
          <h3 className="text-xl font-bold text-gray-900 mb-6 flex items-center space-x-2">
            <span className="text-2xl">🎨</span>
            <span>Visualizations</span>
            <span className="text-sm font-normal text-gray-600 ml-2">({images.length} {images.length === 1 ? 'diagram' : 'diagrams'})</span>
          </h3>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            {images.map((image, index) => (
              <div 
                key={index} 
                className="bg-white rounded-lg shadow-md overflow-hidden transform hover:scale-[1.02] transition-all duration-300 hover:shadow-xl group"
              >
                <div className="relative overflow-hidden bg-gray-100">
                  <img
                    src={image.url}
                    alt={image.alt_text || image.caption || `Visualization ${index + 1}`}
                    className="w-full h-48 object-cover transition-transform duration-300 group-hover:scale-110"
                    onError={(e) => {
                      const target = e.target as HTMLImageElement;
                      target.style.display = 'none';
                      const parent = target.parentElement;
                      if (parent) {
                        parent.innerHTML = `
                          <div class="w-full h-48 flex items-center justify-center bg-gray-200">
                            <p class="text-gray-500 text-sm">Image failed to load</p>
                          </div>
                        `;
                      }
                    }}
                  />
                  <div className="absolute inset-0 bg-gradient-to-t from-black/60 via-black/20 to-transparent opacity-0 group-hover:opacity-100 transition-opacity duration-300"></div>
                  <div className="absolute top-2 right-2 opacity-0 group-hover:opacity-100 transition-opacity duration-300">
                    <button
                      onClick={() => {
                        const link = document.createElement('a');
                        link.href = image.url;
                        link.download = `visualization-${index + 1}.png`;
                        link.target = '_blank';
                        document.body.appendChild(link);
                        link.click();
                        document.body.removeChild(link);
                      }}
                      className="bg-white/90 hover:bg-white text-gray-800 p-2 rounded-full shadow-lg transition-all duration-200 transform hover:scale-110"
                      title="Download diagram"
                    >
                      <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
                      </svg>
                    </button>
                  </div>
                </div>
                <div className="p-4">
                  <p className="text-sm text-gray-700 font-medium line-clamp-2">{image.caption || image.alt_text || `Visualization ${index + 1}`}</p>
                  {image.alt_text && image.alt_text !== image.caption && (
                    <p className="text-xs text-gray-500 mt-1 line-clamp-1">{image.alt_text}</p>
                  )}
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
};

export default LessonCard;
