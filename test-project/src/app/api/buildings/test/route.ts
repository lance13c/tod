import { NextRequest, NextResponse } from 'next/server';
import { getDuckDB } from '@/lib/db/duckdb';
import { ensureDuckDBInitialized } from '@/lib/db/init';

export async function GET(req: NextRequest) {
  try {
    // Initialize DuckDB if needed
    await ensureDuckDBInitialized();
    
    const db = await getDuckDB();
    
    // Test basic DuckDB functionality
    const basicTest = await db.all("SELECT 'Hello DuckDB' as message, 42 as answer");
    
    // Test spatial extension
    let spatialTest = null;
    try {
      spatialTest = await db.all("SELECT ST_Point(1, 2) as point");
    } catch (e) {
      spatialTest = { error: 'Spatial extension not working', details: String(e) };
    }
    
    // Test if table exists
    const tableTest = await db.all(`
      SELECT name FROM sqlite_master 
      WHERE type='table' AND name='buildings'
    `);
    
    // Try a simple insert
    let insertTest = null;
    try {
      await db.run(`
        INSERT INTO buildings (id, name, address, geometry, bbox_minx, bbox_miny, bbox_maxx, bbox_maxy, area, centroid_lon, centroid_lat)
        VALUES ('test_1', 'Test Building', 'Test Address', ST_Point(-86.7816, 36.1627), -86.782, 36.162, -86.781, 36.163, 100, -86.7816, 36.1627)
      `);
      insertTest = { success: true };
      
      // Clean up test data
      await db.run("DELETE FROM buildings WHERE id = 'test_1'");
    } catch (e) {
      insertTest = { error: 'Insert failed', details: String(e) };
    }
    
    // Count buildings
    const count = await db.all('SELECT COUNT(*)::INTEGER as count FROM buildings');
    
    return NextResponse.json({
      basic_test: basicTest[0],
      spatial_test: spatialTest,
      table_exists: tableTest.length > 0,
      insert_test: insertTest,
      building_count: Number(count[0].count)
    });
  } catch (error) {
    console.error('Error testing DuckDB:', error);
    return NextResponse.json(
      { 
        error: 'Failed to test DuckDB',
        details: error instanceof Error ? error.message : 'Unknown error'
      },
      { status: 500 }
    );
  }
}