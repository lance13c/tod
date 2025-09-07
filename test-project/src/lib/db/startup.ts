import { initializeDuckDB } from './duckdb';

/**
 * Initialize database on application startup
 */
export async function initializeDatabase() {
  console.log('🚀 Initializing database on startup...');
  
  try {
    const db = await initializeDuckDB();
    
    // Verify data is loaded
    const result = await db.all('SELECT COUNT(*) as count FROM buildings');
    const count = result[0].count;
    
    if (count === 0) {
      console.warn('⚠️ No buildings in database after initialization');
    } else {
      console.log(`✅ Database ready with ${count} buildings`);
    }
    
    return true;
  } catch (error) {
    console.error('❌ Failed to initialize database:', error);
    return false;
  }
}

// Run initialization immediately when this module is imported
if (typeof process !== 'undefined' && process.env.NODE_ENV !== 'test') {
  initializeDatabase();
}