import { NextRequest, NextResponse } from 'next/server';
import { getDuckDB } from '@/lib/db/duckdb';
import { ensureDuckDBInitialized } from '@/lib/db/init';

export async function POST(req: NextRequest) {
  try {
    const { latitude, longitude, bufferMeters = 100 } = await req.json();
    
    await ensureDuckDBInitialized();
    const db = await getDuckDB();
    
    // Test 1: Simple distance query without spatial functions
    console.log(`Testing query for lat=${latitude}, lon=${longitude}, buffer=${bufferMeters}`);
    
    // Test 2: Find closest building using simple math
    const simpleQuery = `
      SELECT 
        id,
        centroid_lon,
        centroid_lat,
        SQRT(POWER(centroid_lon - ?, 2) + POWER(centroid_lat - ?, 2)) * 111320 as approx_distance_meters
      FROM buildings
      ORDER BY approx_distance_meters
      LIMIT 5
    `;
    
    const simpleResults = await db.all(simpleQuery, longitude, latitude);
    
    // Test 3: Try spatial query
    let spatialResults = null;
    try {
      const spatialQuery = `
        SELECT 
          id,
          centroid_lon,
          centroid_lat,
          ST_Distance_Sphere(ST_Point(centroid_lon, centroid_lat), ST_Point(?, ?)) as distance
        FROM buildings
        WHERE ST_Distance_Sphere(ST_Point(centroid_lon, centroid_lat), ST_Point(?, ?)) <= ?
        ORDER BY distance
        LIMIT 5
      `;
      spatialResults = await db.all(spatialQuery, longitude, latitude, longitude, latitude, bufferMeters);
    } catch (e) {
      spatialResults = { error: String(e) };
    }
    
    return NextResponse.json({
      request: { latitude, longitude, bufferMeters },
      simple_results: simpleResults,
      spatial_results: spatialResults
    });
  } catch (error) {
    console.error('Query test error:', error);
    return NextResponse.json(
      { error: String(error) },
      { status: 500 }
    );
  }
}