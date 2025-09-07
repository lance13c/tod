import { NextRequest, NextResponse } from 'next/server';
import { closeDuckDB, initializeDuckDB } from '@/lib/db/duckdb';

export async function POST(req: NextRequest) {
  try {
    console.log('Resetting DuckDB...');
    
    // Close existing connection
    await closeDuckDB();
    
    // Reinitialize and reload data
    await initializeDuckDB();
    
    return NextResponse.json({
      success: true,
      message: 'DuckDB has been reset and data reloaded'
    });
  } catch (error) {
    console.error('Error resetting DuckDB:', error);
    return NextResponse.json(
      { 
        error: 'Failed to reset DuckDB',
        details: error instanceof Error ? error.message : 'Unknown error'
      },
      { status: 500 }
    );
  }
}