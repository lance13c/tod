#!/usr/bin/env bun

import { PrismaClient } from '@prisma/client';
import fs from 'fs/promises';
import path from 'path';
import { calculateBoundingBox } from '../src/lib/geo/building-detector';

const prisma = new PrismaClient();

interface GeoJSONFeature {
  type: string;
  geometry: {
    type: string;
    coordinates: number[][][];
  };
  properties: {
    release?: number;
    capture_dates_range?: string;
    [key: string]: any;
  };
}

interface GeoJSONCollection {
  type: string;
  features: GeoJSONFeature[];
}

async function loadBuildingData(filePath: string) {
  console.log(`📂 Loading building data from ${filePath}...`);
  
  try {
    const data = await fs.readFile(filePath, 'utf-8');
    const geoJson: GeoJSONCollection = JSON.parse(data);
    
    console.log(`📊 Found ${geoJson.features.length} buildings in file`);
    
    let imported = 0;
    let skipped = 0;
    const batchSize = 100;
    
    // Process in batches to avoid memory issues
    for (let i = 0; i < geoJson.features.length; i += batchSize) {
      const batch = geoJson.features.slice(i, i + batchSize);
      const buildings = [];
      
      for (const feature of batch) {
        if (feature.geometry?.type !== 'Polygon') {
          skipped++;
          continue;
        }
        
        const coordinates = feature.geometry.coordinates[0];
        const bbox = calculateBoundingBox(coordinates);
        
        // Calculate approximate area (simplified)
        const latDiff = bbox.maxLat - bbox.minLat;
        const lngDiff = bbox.maxLng - bbox.minLng;
        const approxArea = latDiff * lngDiff * 111000 * 111000; // Very rough approximation in m²
        
        buildings.push({
          polygon: JSON.stringify(feature.geometry),
          bbox: JSON.stringify(bbox),
          area: approxArea,
          // Tennessee-specific data
          name: feature.properties.name || null,
          address: feature.properties.address || null,
          osmId: feature.properties.osm_id || null,
        });
      }
      
      if (buildings.length > 0) {
        // Try to create buildings, catch duplicates
        try {
          await prisma.building.createMany({
            data: buildings,
          });
          imported += buildings.length;
        } catch (error: any) {
          // If it's a unique constraint error, try inserting one by one
          if (error?.code === 'P2002') {
            for (const building of buildings) {
              try {
                await prisma.building.create({
                  data: building,
                });
                imported++;
              } catch (e) {
                // Skip duplicates
                skipped++;
              }
            }
          } else {
            throw error;
          }
        }
      }
      
      // Progress update
      const progress = Math.min(100, Math.round((i + batchSize) / geoJson.features.length * 100));
      console.log(`⏳ Progress: ${progress}% (${imported} imported, ${skipped} skipped)`);
    }
    
    console.log(`\n✅ Import complete!`);
    console.log(`📊 Summary:`);
    console.log(`   - Total features: ${geoJson.features.length}`);
    console.log(`   - Imported: ${imported}`);
    console.log(`   - Skipped: ${skipped}`);
    
  } catch (error) {
    console.error('❌ Error loading building data:', error);
    throw error;
  }
}

// Main execution
async function main() {
  const args = process.argv.slice(2);
  
  if (args.length === 0) {
    console.log('Usage: bun run scripts/load-building-data.ts <path-to-geojson>');
    console.log('Example: bun run scripts/load-building-data.ts ~/Desktop/Tennessee.geojson');
    process.exit(1);
  }
  
  const filePath = path.resolve(args[0]);
  
  // Check if file exists
  try {
    await fs.access(filePath);
  } catch {
    console.error(`❌ File not found: ${filePath}`);
    process.exit(1);
  }
  
  try {
    await loadBuildingData(filePath);
    await prisma.$disconnect();
    console.log('🎉 Done!');
  } catch (error) {
    console.error('Fatal error:', error);
    await prisma.$disconnect();
    process.exit(1);
  }
}

main();