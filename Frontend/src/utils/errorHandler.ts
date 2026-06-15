// Global error handler to suppress autofill extension conflicts

// List of error patterns to suppress
const SUPPRESSED_ERRORS = [
  'Cannot read properties of null (reading \'username\')',
  'AutofillOverlayContentService',
  'bootstrap-autofill-overlay-notifications.js',
  'getFormFieldData',
  'handleFormFieldSubmitEvent',
  'handleSubmitButtonInteraction'
]

// Original console methods
const originalConsoleError = console.error;
const originalConsoleWarn = console.warn;

/**
 * Check if an error should be suppressed
 */
function shouldSuppressError(error: unknown): boolean {
  const errorString = typeof error === 'string'
    ? error
    : error instanceof Error
      ? error.message
      : String(error ?? '');
  
  return SUPPRESSED_ERRORS.some(pattern => 
    errorString.includes(pattern)
  );
}

/**
 * Enhanced console.error that filters out autofill extension errors
 */
function filteredConsoleError(...args: unknown[]) {
  // Check if any argument contains a suppressed error pattern
  const shouldSuppress = args.some(arg => shouldSuppressError(arg));
  
  if (!shouldSuppress) {
    originalConsoleError.apply(console, args);
  }
}

/**
 * Enhanced console.warn that filters out autofill extension warnings
 */
function filteredConsoleWarn(...args: unknown[]) {
  // Check if any argument contains a suppressed error pattern
  const shouldSuppress = args.some(arg => shouldSuppressError(arg));
  
  if (!shouldSuppress) {
    originalConsoleWarn.apply(console, args);
  }
}

/**
 * Global error event handler
 */
function globalErrorHandler(event: ErrorEvent) {
  if (shouldSuppressError(event.error)) {
    event.preventDefault();
    event.stopPropagation();
    return false;
  }
}

/**
 * Unhandled promise rejection handler
 */
function unhandledRejectionHandler(event: PromiseRejectionEvent) {
  if (shouldSuppressError(event.reason)) {
    event.preventDefault();
    event.stopPropagation();
    return false;
  }
}

/**
 * Initialize error handling to suppress autofill extension conflicts
 */
export function initializeErrorHandling() {
  // Only run in browser environment
  if (typeof window === 'undefined') {
    return;
  }

  // Override console methods
  console.error = filteredConsoleError;
  console.warn = filteredConsoleWarn;

  // Add global event listeners
  window.addEventListener('error', globalErrorHandler);
  window.addEventListener('unhandledrejection', unhandledRejectionHandler);

  // Log that error handling is initialized (only in development)
  if (process.env.NODE_ENV === 'development') {
    originalConsoleLog('🔧 Autofill extension error handling initialized');
  }
}

/**
 * Restore original error handling (useful for testing)
 */
export function restoreErrorHandling() {
  if (typeof window === 'undefined') {
    return;
  }

  // Restore original console methods
  console.error = originalConsoleError;
  console.warn = originalConsoleWarn;

  // Remove global event listeners
  window.removeEventListener('error', globalErrorHandler);
  window.removeEventListener('unhandledrejection', unhandledRejectionHandler);

  if (process.env.NODE_ENV === 'development') {
    originalConsoleLog('🔧 Original error handling restored');
  }
}

// Store original console.log for our own logging
const originalConsoleLog = console.log;

// Export for testing purposes
export { shouldSuppressError, originalConsoleError, originalConsoleWarn };
