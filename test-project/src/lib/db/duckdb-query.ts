import { Database } from 'duckdb-async';
import path from 'path';

let queryDb: Database | null = null;

/**
 * Get a read-only DuckDB connection for queries
 * This avoids lock conflicts with the main write connection
 */
export async function getQueryDatabase(): Promise<Database> {
  if (queryDb) {
    return queryDb;
  }

  try {
    const dbPath = path.join(process.cwd(), '.duckdb', 'buildings.db');
    
    // Create a read-only connection
    queryDb = await Database.create(dbPath, { 
      access_mode: 'read_only'
    });
    
    // Load spatial extension for queries
    await queryDb.run('LOAD spatial');
    
    console.log('Query database connection established (read-only)');
    return queryDb;
  } catch (error) {
    console.error('Failed to create query database:', error);
    // Fallback to using the main connection
    const { initializeDuckDB } = await import('./duckdb');
    return initializeDuckDB();
  }
}

/**
 * Execute a spatial query using the read-only connection
 */
export async function spatialQuery<T = any>(query: string, params: any[] = []): Promise<T[]> {
  const db = await getQueryDatabase();
  return db.all(query, ...params) as Promise<T[]>;
}