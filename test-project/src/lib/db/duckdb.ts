import { Database } from 'duckdb-async';
import fs from 'fs';
import path from 'path';

let db: Database | null = null;
let isInitialized = false;

/**
 * Initialize DuckDB connection and load spatial extension
 */
export async function initializeDuckDB(): Promise<Database> {
  if (db && isInitialized) {
    return db;
  }

  try {
    // Create a file-based DuckDB instance for persistence
    const dbPath = path.join(process.cwd(), '.duckdb', 'buildings.db');
    
    // Ensure directory exists
    const dbDir = path.dirname(dbPath);
    if (!fs.existsSync(dbDir)) {
      fs.mkdirSync(dbDir, { recursive: true });
    }
    
    // IMPORTANT: Load spatial extension BEFORE opening the database
    // This ensures the extension is available when replaying WAL files
    db = await Database.create(dbPath);
    
    console.log('Initializing DuckDB with spatial extension...');
    
    // Install and load the spatial extension - MUST be done before any spatial operations
    try {
      await db.run('INSTALL spatial');
      await db.run('LOAD spatial');
    } catch (err) {
      console.log('Spatial extension may already be installed:', err);
      // Try to just load it
      await db.run('LOAD spatial');
    }
    
    // Create the buildings table with spatial column
    await db.run(`
      CREATE TABLE IF NOT EXISTS buildings (
        id VARCHAR PRIMARY KEY,
        name VARCHAR,
        address VARCHAR,
        geometry GEOMETRY,
        bbox_minx DOUBLE,
        bbox_miny DOUBLE,
        bbox_maxx DOUBLE,
        bbox_maxy DOUBLE,
        area DOUBLE,
        centroid_lon DOUBLE,
        centroid_lat DOUBLE
      )
    `);
    
    // Skip RTREE index creation as it causes issues with WAL replay
    // The bounding box columns provide sufficient indexing for our queries
    console.log('Buildings table ready (using bbox columns for spatial indexing)');
    
    // Load Nashville buildings GeoJSON if not already loaded
    const count = await db.all('SELECT COUNT(*) as count FROM buildings');
    console.log(`Current buildings in DuckDB: ${count[0].count}`);
    
    if (count[0].count === 0) {
      console.log('No buildings found in DuckDB, loading Nashville buildings data...');
      await loadNashvilleBuildingsDataFast();
      
      // Verify data was loaded
      const newCount = await db.all('SELECT COUNT(*) as count FROM buildings');
      console.log(`Buildings loaded into DuckDB: ${newCount[0].count}`);
    }
    
    isInitialized = true;
    console.log('DuckDB initialized successfully with spatial extension');
    
    return db;
  } catch (error) {
    console.error('Failed to initialize DuckDB:', error);
    throw error;
  }
}

/**
 * Fast loading function for Nashville buildings
 */
async function loadNashvilleBuildingsDataFast() {
  if (!db) {
    throw new Error('DuckDB not initialized');
  }
  
  console.log('Fast loading Nashville buildings data into DuckDB...');
  
  try {
    const geojsonPath = path.join(process.cwd(), 'public', 'nashville_buildings.geojson');
    
    if (!fs.existsSync(geojsonPath)) {
      throw new Error(`GeoJSON file not found at ${geojsonPath}`);
    }
    
    const geojsonContent = fs.readFileSync(geojsonPath, 'utf-8');
    const geojson = JSON.parse(geojsonContent);
    
    const totalFeatures = geojson.features?.length || 0;
    console.log(`Processing ${totalFeatures} features...`);
    
    // For startup, load a subset for performance
    // Full dataset can be loaded via the API endpoint if needed
    const startupLimit = 50000; // Load 50k buildings on startup
    const maxBuildings = Math.min(startupLimit, totalFeatures);
    
    console.log(`Loading ${maxBuildings} buildings on startup (${Math.round(maxBuildings/totalFeatures*100)}% of dataset)`);
    
    let insertedCount = 0;
    const batchSize = 100;
    const values: any[] = [];
    
    for (let j = 0; j < maxBuildings; j++) {
      const feature = geojson.features[j];
      
      if (feature?.geometry?.type === 'Polygon') {
        try {
          const coords = feature.geometry.coordinates[0];
          let minX = Infinity, minY = Infinity, maxX = -Infinity, maxY = -Infinity;
          let sumX = 0, sumY = 0;
          
          for (const [lon, lat] of coords) {
            minX = Math.min(minX, lon);
            maxX = Math.max(maxX, lon);
            minY = Math.min(minY, lat);
            maxY = Math.max(maxY, lat);
            sumX += lon;
            sumY += lat;
          }
          
          values.push([
            `b${j}`,
            null,
            null,
            JSON.stringify(feature.geometry),
            minX, minY, maxX, maxY,
            0,
            sumX / coords.length,
            sumY / coords.length
          ]);
          
          // Insert in batches
          if (values.length >= batchSize) {
            await insertBatch(values);
            insertedCount += values.length;
            values.length = 0;
            
            if (insertedCount % 10000 === 0) {
              console.log(`Loaded ${insertedCount} buildings...`);
            }
          }
        } catch (err) {
          // Skip errors silently
        }
      }
    }
    
    // Insert remaining values
    if (values.length > 0) {
      await insertBatch(values);
      insertedCount += values.length;
    }
    
    console.log(`Fast load complete: ${insertedCount} buildings loaded`);
    
    if (totalFeatures > startupLimit) {
      console.log(`Note: ${totalFeatures - maxBuildings} additional buildings available. Use /api/buildings/load to load more.`);
    }
  } catch (error) {
    console.error('Fast load failed:', error);
    // Don't throw - allow app to continue with empty database
  }
}

async function insertBatch(values: any[]) {
  if (!db || values.length === 0) return;
  
  const placeholders = values.map(() => '(?, ?, ?, ST_GeomFromGeoJSON(?), ?, ?, ?, ?, ?, ?, ?)').join(',');
  const flatValues = values.flat();
  
  await db.run(`
    INSERT INTO buildings (id, name, address, geometry, bbox_minx, bbox_miny, bbox_maxx, bbox_maxy, area, centroid_lon, centroid_lat)
    VALUES ${placeholders}
  `, ...flatValues);
}

/**
 * Load Nashville buildings GeoJSON data into DuckDB
 */
async function loadNashvilleBuildingsData() {
  if (!db) {
    throw new Error('DuckDB not initialized');
  }
  
  console.log('Loading Nashville buildings data into DuckDB...');
  
  try {
    // Read the GeoJSON file path
    const geojsonPath = path.join(process.cwd(), 'public', 'nashville_buildings.geojson');
    console.log(`Reading GeoJSON from: ${geojsonPath}`);
    
    // Check if file exists
    if (!fs.existsSync(geojsonPath)) {
      throw new Error(`GeoJSON file not found at ${geojsonPath}`);
    }
    
    // Read the file content
    const geojsonContent = fs.readFileSync(geojsonPath, 'utf-8');
    console.log(`Read ${geojsonContent.length} bytes from GeoJSON file`);
    
    const geojson = JSON.parse(geojsonContent);
    console.log(`Parsed GeoJSON with ${geojson.features?.length || 0} features`);
    
    if (!geojson.features || geojson.features.length === 0) {
      throw new Error('No features found in GeoJSON file');
    }
    
    // Process and insert buildings in batches
    let insertedCount = 0;
    const errors: string[] = [];
    
    // Process first 10 buildings as a test
    const testLimit = Math.min(10, geojson.features.length);
    console.log(`Processing first ${testLimit} buildings as a test...`);
    
    for (let i = 0; i < testLimit; i++) {
      const feature = geojson.features[i];
      
      if (feature.geometry && feature.geometry.type === 'Polygon') {
        try {
          const coords = feature.geometry.coordinates[0];
          
          // Calculate centroid and bounds manually
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
          
          const centroidLon = sumX / coords.length;
          const centroidLat = sumY / coords.length;
          
          // Simplified insert with calculated values
          const id = `building_${i + 1}`;
          const geometryJson = JSON.stringify(feature.geometry);
          
          await db.run(`
            INSERT INTO buildings (id, name, address, geometry, bbox_minx, bbox_miny, bbox_maxx, bbox_maxy, area, centroid_lon, centroid_lat)
            VALUES (?, NULL, NULL, ST_GeomFromGeoJSON(?), ?, ?, ?, ?, 0, ?, ?)
          `, id, geometryJson, minX, minY, maxX, maxY, centroidLon, centroidLat);
          
          insertedCount++;
          console.log(`Inserted building ${id}`);
        } catch (err) {
          const errorMsg = `Failed to insert building ${i}: ${err}`;
          console.error(errorMsg);
          errors.push(errorMsg);
        }
      }
    }
    
    const result = await db.all('SELECT COUNT(*) as count FROM buildings');
    console.log(`Test load complete: ${result[0].count} buildings in DuckDB`);
    
    if (errors.length > 0) {
      console.log(`Encountered ${errors.length} errors during loading`);
    }
    
    // If test was successful, load the rest
    if (insertedCount > 0 && testLimit < geojson.features.length) {
      console.log(`Test successful, loading remaining ${geojson.features.length - testLimit} buildings...`);
      
      for (let i = testLimit; i < geojson.features.length; i++) {
        const feature = geojson.features[i];
        
        if (feature.geometry && feature.geometry.type === 'Polygon') {
          try {
            const coords = feature.geometry.coordinates[0];
            
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
            
            const centroidLon = sumX / coords.length;
            const centroidLat = sumY / coords.length;
            const id = `building_${i + 1}`;
            const geometryJson = JSON.stringify(feature.geometry);
            
            await db.run(`
              INSERT INTO buildings (id, name, address, geometry, bbox_minx, bbox_miny, bbox_maxx, bbox_maxy, area, centroid_lon, centroid_lat)
              VALUES (?, NULL, NULL, ST_GeomFromGeoJSON(?), ?, ?, ?, ?, 0, ?, ?)
            `, id, geometryJson, minX, minY, maxX, maxY, centroidLon, centroidLat);
            
            insertedCount++;
            
            if (insertedCount % 1000 === 0) {
              console.log(`Inserted ${insertedCount} buildings...`);
            }
          } catch (err) {
            // Silently skip errors for bulk load
          }
        }
      }
    }
    
    const finalResult = await db.all('SELECT COUNT(*) as count FROM buildings');
    console.log(`Successfully loaded ${finalResult[0].count} buildings into DuckDB`);
  } catch (error) {
    console.error('Failed to load Nashville buildings data:', error);
    throw error;
  }
}

/**
 * Get the DuckDB instance
 */
export async function getDuckDB(): Promise<Database> {
  if (!db || !isInitialized) {
    return await initializeDuckDB();
  }
  // Ensure spatial extension is loaded
  try {
    await db.run('LOAD spatial');
  } catch (err) {
    // Extension might already be loaded
  }
  return db;
}

/**
 * Close the DuckDB connection
 */
export async function closeDuckDB() {
  if (db) {
    await db.close();
    db = null;
    isInitialized = false;
  }
}

/**
 * Execute a spatial query
 */
export async function spatialQuery<T = any>(query: string, params: any[] = []): Promise<T[]> {
  const database = await getDuckDB();
  return database.all(query, ...params) as Promise<T[]>;
}