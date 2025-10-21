import React, { useState } from 'react';
import { OGLesson, ImageRef, PDFResponse } from '../types';

interface LessonCardProps {
  lesson: OGLesson;
  images?: ImageRef[];
  sessionId?: string;
}

const LessonCard: React.FC<LessonCardProps> = ({ lesson, images = [], sessionId }) => {
  const [isGeneratingPDF, setIsGeneratingPDF] = useState(false);
  const [pdfError, setPdfError] = useState<string | null>(null);

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
    <div className="space-y-6">
      {/* PDF Download Button */}
      {sessionId && (
        <div className="flex justify-end mb-6">
          <button
            onClick={handleDownloadPDF}
            disabled={isGeneratingPDF}
            className="bg-red-600 text-white px-4 py-2 rounded-lg font-medium hover:bg-red-700 focus:ring-2 focus:ring-red-500 focus:ring-offset-2 disabled:opacity-50 disabled:cursor-not-allowed transition-colors flex items-center space-x-2"
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

      {sections.map((section) => (
        <div key={section.key} className={`border-l-4 pl-4 ${section.color}`}>
          <h3 className="text-lg font-semibold text-gray-900 mb-2">{section.title}</h3>
          {section.isCode ? (
            <pre className="bg-gray-100 p-3 rounded text-sm overflow-x-auto">
              <code>{section.content}</code>
            </pre>
          ) : (
            <p className="text-gray-700">{section.content}</p>
          )}
        </div>
      ))}

      {/* Visualizations */}
      {images && images.length > 0 && (
        <div>
          <h3 className="text-lg font-semibold text-gray-900 mb-4">Visualizations</h3>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            {images.map((image, index) => (
              <div key={index} className="border rounded-lg p-4">
                <img
                  src={image.url}
                  alt={image.alt_text}
                  className="w-full h-48 object-cover rounded mb-2"
                  onError={(e) => {
                    const target = e.target as HTMLImageElement;
                    target.style.display = 'none';
                  }}
                />
                <p className="text-sm text-gray-600">{image.caption}</p>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
};

export default LessonCard;
