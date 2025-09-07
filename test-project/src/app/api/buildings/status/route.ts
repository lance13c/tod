import { NextRequest, NextResponse } from 'next/server';
import { getDuckDB } from '@/lib/db/duckdb';

export async function GET(req: NextRequest) {
  try {
    const db = await getDuckDB();
    
    // Get count and bounds
    const countResult = await db.all('SELECT COUNT(*)::INTEGER as count FROM buildings');
    const boundsResult = await db.all(`
      SELECT 
        CAST(MIN(bbox_minx) AS DOUBLE) as min_lon,
        CAST(MAX(bbox_maxx) AS DOUBLE) as max_lon,
        CAST(MIN(bbox_miny) AS DOUBLE) as min_lat,
        CAST(MAX(bbox_maxy) AS DOUBLE) as max_lat
      FROM buildings
    `);
    
    const count = Number(countResult[0].count);
    const bounds = boundsResult[0];
    
    return NextResponse.json({
      status: count > 0 ? 'ready' : 'empty',
      total_buildings: count,
      bounds: bounds,
      message: count > 0 
        ? `Database ready with ${count} buildings` 
        : 'No buildings loaded - restart the server to load data'
    });
  } catch (error) {
    console.error('Error checking DuckDB status:', error);
    return NextResponse.json(
      { 
        status: 'error',
        error: 'Failed to check database status',
        details: error instanceof Error ? error.message : 'Unknown error'
      },
      { status: 500 }
    );
  }
}