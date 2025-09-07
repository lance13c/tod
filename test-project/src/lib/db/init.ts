// Initialize DuckDB on server startup
// This will be imported by API routes that need spatial queries
let initPromise: Promise<void> | null = null;

export async function ensureDuckDBInitialized() {
  // Skip initialization during build
  if (process.env.NEXT_PHASE === 'phase-production-build') {
    return;
  }
  
  if (!initPromise) {
    initPromise = import('./duckdb').then(({ initializeDuckDB }) => {
      return initializeDuckDB().then(() => {
        console.log('DuckDB initialized and ready for spatial queries');
      });
    }).catch(error => {
      console.error('Failed to initialize DuckDB:', error);
      // Reset promise so it can be retried
      initPromise = null;
      throw error;
    });
  }
  return initPromise;
}