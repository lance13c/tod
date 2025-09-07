import { NextRequest, NextResponse } from 'next/server';
import { ensureDuckDBInitialized } from '@/lib/db/init';
import { getDuckDB } from '@/lib/db/duckdb';

export async function GET(req: NextRequest) {
  try {
    // Initialize DuckDB if needed
    await ensureDuckDBInitialized();
    
    const db = await getDuckDB();
    
    // Get statistics about the buildings table
    const count = await db.all('SELECT COUNT(*)::INTEGER as count FROM buildings');
    const sample = await db.all('SELECT id, CAST(centroid_lon AS DOUBLE) as centroid_lon, CAST(centroid_lat AS DOUBLE) as centroid_lat, CAST(area AS DOUBLE) as area FROM buildings LIMIT 5');
    const bounds = await db.all(`
      SELECT 
        CAST(MIN(bbox_minx) AS DOUBLE) as min_lon,
        CAST(MAX(bbox_maxx) AS DOUBLE) as max_lon,
        CAST(MIN(bbox_miny) AS DOUBLE) as min_lat,
        CAST(MAX(bbox_maxy) AS DOUBLE) as max_lat
      FROM buildings
    `);
    
    // Convert any BigInt values to strings for JSON serialization
    const serializableCount = Number(count[0].count);
    
    return NextResponse.json({
      total_buildings: serializableCount,
      bounds: bounds[0],
      sample_buildings: sample,
      status: 'DuckDB is running and data is loaded'
    });
  } catch (error) {
    console.error('Error debugging DuckDB:', error);
    return NextResponse.json(
      { 
        error: 'Failed to debug DuckDB',
        details: error instanceof Error ? error.message : 'Unknown error'
      },
      { status: 500 }
    );
  }
}