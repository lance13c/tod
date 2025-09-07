import { NextRequest, NextResponse } from 'next/server';
import { getDuckDB } from '@/lib/db/duckdb';
import { ensureDuckDBInitialized } from '@/lib/db/init';
import fs from 'fs';
import path from 'path';

export async function POST(req: NextRequest) {
  try {
    // Initialize DuckDB if needed
    await ensureDuckDBInitialized();
    
    const db = await getDuckDB();
    
    // Clear existing data
    await db.run('DELETE FROM buildings');
    console.log('Cleared existing buildings');
    
    // Read the GeoJSON file
    const geojsonPath = path.join(process.cwd(), 'public', 'nashville_buildings.geojson');
    
    if (!fs.existsSync(geojsonPath)) {
      return NextResponse.json({ error: 'GeoJSON file not found' }, { status: 404 });
    }
    
    const geojsonContent = fs.readFileSync(geojsonPath, 'utf-8');
    const geojson = JSON.parse(geojsonContent);
    
    console.log(`Loading ${geojson.features?.length || 0} features`);
    
    let successCount = 0;
    let errorCount = 0;
    const errors: string[] = [];
    
    // Load all buildings (or specify a limit in the request)
    const requestBody = await req.json().catch(() => ({}));
    const requestLimit = requestBody.limit;
    const loadAll = requestBody.all === true;
    
    // Determine how many to load
    const limit = loadAll ? geojson.features.length : (requestLimit || 50000);
    
    console.log(`Loading ${limit} buildings (${loadAll ? 'ALL' : 'LIMITED'})...`);
    
    // Batch insert for better performance
    const batchSize = 100;
    const values: any[] = [];
    
    for (let i = 0; i < limit && i < geojson.features.length; i++) {
      const feature = geojson.features[i];
      
      if (feature.geometry && feature.geometry.type === 'Polygon') {
        try {
          const coords = feature.geometry.coordinates[0];
          
          // Calculate bounds and centroid
          let minX = Infinity, minY = Infinity, maxX = -Infinity, maxY = -Infinity;
          let sumX = 0, sumY = 0;
          
          for (const coord of coords) {
            const [lon, lat] = coord;
            minX = Math.min(minX, lon);
            maxX = Math.max(maxX, lon);
            minY = Math.min(minY, lat);
            maxY = Math.max(maxY, lat);
            sumX += lon;
            sumY += lat;
          }
          
          values.push([
            `bldg_${i}`,
            null,
            null,
            JSON.stringify(feature.geometry),
            minX, minY, maxX, maxY,
            0,
            sumX / coords.length,
            sumY / coords.length
          ]);
          
          // Insert when batch is full
          if (values.length >= batchSize) {
            const placeholders = values.map(() => '(?, ?, ?, ST_GeomFromGeoJSON(?), ?, ?, ?, ?, ?, ?, ?)').join(',');
            const flatValues = values.flat();
            
            await db.run(`
              INSERT INTO buildings (id, name, address, geometry, bbox_minx, bbox_miny, bbox_maxx, bbox_maxy, area, centroid_lon, centroid_lat)
              VALUES ${placeholders}
            `, ...flatValues);
            
            successCount += values.length;
            values.length = 0;
            
            if (successCount % 10000 === 0) {
              console.log(`Loaded ${successCount} buildings...`);
            }
          }
        } catch (err) {
          errorCount++;
          if (errorCount <= 5) {
            errors.push(`Building ${i}: ${err}`);
          }
        }
      }
    }
    
    // Insert remaining values
    if (values.length > 0) {
      const placeholders = values.map(() => '(?, ?, ?, ST_GeomFromGeoJSON(?), ?, ?, ?, ?, ?, ?, ?)').join(',');
      const flatValues = values.flat();
      
      await db.run(`
        INSERT INTO buildings (id, name, address, geometry, bbox_minx, bbox_miny, bbox_maxx, bbox_maxy, area, centroid_lon, centroid_lat)
        VALUES ${placeholders}
      `, ...flatValues);
      
      successCount += values.length;
    }
    
    // Get final count
    const count = await db.all('SELECT COUNT(*)::INTEGER as count FROM buildings');
    
    return NextResponse.json({
      success: true,
      loaded: successCount,
      errors: errorCount,
      error_samples: errors,
      total_in_db: Number(count[0].count),
      message: `Loaded ${successCount} out of ${limit} buildings`
    });
  } catch (error) {
    console.error('Error loading buildings:', error);
    return NextResponse.json(
      { 
        error: 'Failed to load buildings',
        details: error instanceof Error ? error.message : 'Unknown error'
      },
      { status: 500 }
    );
  }
}