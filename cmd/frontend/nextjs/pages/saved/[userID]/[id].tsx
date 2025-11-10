import { useState, useEffect } from 'react';
import { useRouter } from 'next/router';
import Head from 'next/head';
import LessonCard from '../../../components/LessonCard';
import { OGLesson, ImageRef } from '../../../types';
import { getOrchestratorURL } from '../../../utils/getOrchestratorURL';

interface SavedLesson {
  id: string;
  session_id: string;
  user_id: string;
  topic: string;
  title: string;
  explanation_type: string;
  result: {
    lesson: OGLesson;
    images: Record<string, string>;
    summary: string;
  };
  created_at: string;
  updated_at: string;
}

export default function SavedLessonPage() {
  const router = useRouter();
  const { userID, id } = router.query;
  const [savedLesson, setSavedLesson] = useState<SavedLesson | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!userID || !id) return;

    const fetchSavedLesson = async () => {
      setIsLoading(true);
      setError(null);

      try {
        const orchestratorURL = getOrchestratorURL();
        const response = await fetch(`${orchestratorURL}/api/saved/${userID}/${id}`);

        if (!response.ok) {
          if (response.status === 404) {
            throw new Error('Saved lesson not found');
          }
          throw new Error('Failed to load saved lesson');
        }

        const data = await response.json();
        
        // Parse lesson from JSON string if needed
        if (data.result && data.result.lesson && typeof data.result.lesson === 'string') {
          try {
            data.result.lesson = JSON.parse(data.result.lesson);
          } catch (parseError) {
            console.error('Failed to parse lesson JSON:', parseError);
            // If parsing fails, try to create a minimal lesson object
            data.result.lesson = {
              big_picture: data.result.summary || '',
              metaphor: '',
              core_mechanism: '',
              toy_example_code: '',
              memory_hook: '',
              real_life: '',
              best_practices: '',
            };
          }
        }
        
        setSavedLesson(data);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to load saved lesson');
      } finally {
        setIsLoading(false);
      }
    };

    fetchSavedLesson();
  }, [userID, id]);

  const handleDelete = async () => {
    if (!userID || !id || !confirm('Are you sure you want to delete this saved lesson?')) {
      return;
    }

    try {
      const orchestratorURL = process.env.NEXT_PUBLIC_ORCHESTRATOR_URL || 'http://localhost:8080';
      const response = await fetch(`${orchestratorURL}/api/saved/${userID}/${id}`, {
        method: 'DELETE',
      });

      if (!response.ok) {
        throw new Error('Failed to delete saved lesson');
      }

      router.push('/');
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Failed to delete saved lesson');
    }
  };

  if (isLoading) {
    return (
      <>
        <Head>
          <title>Loading Saved Lesson - ExplainIQ</title>
        </Head>
        <div className="min-h-screen bg-gray-50 flex items-center justify-center">
          <div className="text-center">
            <svg className="animate-spin h-12 w-12 text-blue-600 mx-auto mb-4" fill="none" viewBox="0 0 24 24">
              <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
              <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
            </svg>
            <p className="text-gray-600">Loading saved lesson...</p>
          </div>
        </div>
      </>
    );
  }

  if (error || !savedLesson) {
    return (
      <>
        <Head>
          <title>Error - ExplainIQ</title>
        </Head>
        <div className="min-h-screen bg-gray-50 flex items-center justify-center">
          <div className="text-center max-w-md mx-auto p-6">
            <div className="bg-red-50 border border-red-200 rounded-lg p-6">
              <svg className="h-12 w-12 text-red-400 mx-auto mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
              <h2 className="text-xl font-semibold text-red-900 mb-2">Error</h2>
              <p className="text-red-700 mb-4">{error || 'Saved lesson not found'}</p>
              <button
                onClick={() => router.push('/')}
                className="bg-blue-600 text-white px-4 py-2 rounded-lg hover:bg-blue-700 transition-colors"
              >
                Go Home
              </button>
            </div>
          </div>
        </div>
      </>
    );
  }

  // Convert images from Record<string, string> to ImageRef[]
  const images: ImageRef[] = savedLesson.result?.images
    ? Object.entries(savedLesson.result.images).map(([key, url], idx) => ({
        url,
        alt_text: `Image ${idx + 1}`,
        caption: `Image ${idx + 1}`,
      }))
    : [];

  // Ensure lesson is parsed and available
  const lesson = savedLesson.result?.lesson || {
    big_picture: savedLesson.result?.summary || '',
    metaphor: '',
    core_mechanism: '',
    toy_example_code: '',
    memory_hook: '',
    real_life: '',
    best_practices: '',
  };

  return (
    <>
      <Head>
        <title>{savedLesson.title || savedLesson.topic} - ExplainIQ</title>
      </Head>

      <div className="min-h-screen bg-gray-50">
        {/* Header */}
        <div className="bg-white border-b border-gray-200 shadow-sm">
          <div className="container mx-auto px-4 py-4 max-w-5xl">
            <div className="flex items-center justify-between">
              <div>
                <button
                  onClick={() => router.push('/')}
                  className="text-gray-600 hover:text-gray-900 mb-2 flex items-center space-x-2"
                >
                  <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 19l-7-7m0 0l7-7m-7 7h18" />
                  </svg>
                  <span>Back to Home</span>
                </button>
                <h1 className="text-2xl font-bold text-gray-900">{savedLesson.title || savedLesson.topic}</h1>
                <p className="text-sm text-gray-500 mt-1">
                  Saved on {new Date(savedLesson.created_at).toLocaleString()}
                </p>
              </div>
              <button
                onClick={handleDelete}
                className="text-red-600 hover:text-red-700 px-4 py-2 rounded-lg hover:bg-red-50 transition-colors flex items-center space-x-2"
              >
                <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                </svg>
                <span>Delete</span>
              </button>
            </div>
          </div>
        </div>

        {/* Content */}
        <div className="container mx-auto px-4 py-8 max-w-5xl">
          <LessonCard
            lesson={lesson}
            images={images}
            sessionId={savedLesson.session_id}
            explanationType={savedLesson.explanation_type}
            topic={savedLesson.topic}
          />
        </div>
      </div>
    </>
  );
}

