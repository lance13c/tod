import { NextRequest, NextResponse } from 'next/server';
import { writeFile, mkdir } from 'fs/promises';
import path from 'path';
import { prisma as db } from '@/server/db';

// Configure max body size for file uploads (250MB)
export const runtime = 'nodejs';
export const maxDuration = 60; // 60 seconds timeout for large uploads

export async function POST(request: NextRequest) {
  try {
    console.log('Upload API called');
    console.log('Content-Type:', request.headers.get('content-type'));
    
    const formData = await request.formData();
    console.log('FormData parsed successfully');
    
    const file = formData.get('file') as File;
    const sessionId = formData.get('sessionId') as string;
    const participantId = formData.get('participantId') as string;
    
    console.log('File:', file?.name, 'Size:', file?.size);
    console.log('Session ID:', sessionId);
    console.log('Participant ID:', participantId);

    if (!file || !sessionId || !participantId) {
      return NextResponse.json(
        { error: 'Missing required fields' },
        { status: 400 }
      );
    }

    // Verify session exists and participant is in it
    const session = await db.shareSession.findUnique({
      where: { id: sessionId },
      include: {
        participants: true
      }
    });

    if (!session) {
      return NextResponse.json(
        { error: 'Session not found' },
        { status: 404 }
      );
    }

    const participant = session.participants.find(p => p.id === participantId);
    if (!participant) {
      return NextResponse.json(
        { error: 'You are not a participant in this session' },
        { status: 403 }
      );
    }

    // Create directory for session if it doesn't exist
    const uploadsDir = path.join(process.cwd(), 'uploads', sessionId);
    await mkdir(uploadsDir, { recursive: true });

    // Generate unique filename to prevent collisions
    const timestamp = Date.now();
    const fileName = `${timestamp}-${file.name}`;
    const filePath = path.join(uploadsDir, fileName);

    // Convert file to buffer and save
    const bytes = await file.arrayBuffer();
    const buffer = Buffer.from(bytes);
    await writeFile(filePath, buffer);

    // Save file metadata to database
    const document = await db.document.create({
      data: {
        originalName: file.name,
        filename: fileName,
        mimetype: file.type || 'application/octet-stream',
        size: file.size,
        storageType: 'local',
        path: filePath,
        sessionId: sessionId,
        transferStatus: 'completed',
        transferProgress: 100,
        // Note: uploadedById would be for User, but we're using guest participants
        // So we'll leave uploaderId null for now
      }
    });

    // Update session's updated timestamp
    await db.shareSession.update({
      where: { id: sessionId },
      data: { updatedAt: new Date() }
    });

    return NextResponse.json({
      success: true,
      document: {
        id: document.id,
        originalName: document.originalName,
        filename: document.filename,
        mimetype: document.mimetype,
        size: document.size,
        uploadedAt: document.createdAt
      }
    });

  } catch (error) {
    console.error('Upload error:', error);
    // Return detailed error for debugging
    return NextResponse.json(
      { 
        error: 'Failed to upload file',
        details: error instanceof Error ? error.message : 'Unknown error',
        stack: error instanceof Error ? error.stack : undefined
      },
      { status: 500 }
    );
  }
}

// Serve files
export async function GET(request: NextRequest) {
  try {
    const { searchParams } = new URL(request.url);
    const sessionId = searchParams.get('sessionId');
    const filename = searchParams.get('filename');

    if (!sessionId || !filename) {
      return NextResponse.json(
        { error: 'Missing required parameters' },
        { status: 400 }
      );
    }

    // Verify file exists in database
    const document = await db.document.findFirst({
      where: {
        sessionId: sessionId,
        filename: filename
      }
    });

    if (!document) {
      return NextResponse.json(
        { error: 'File not found' },
        { status: 404 }
      );
    }

    // Read file from disk
    const { readFile } = await import('fs/promises');
    const filePath = path.join(process.cwd(), 'uploads', sessionId, filename);
    
    try {
      const fileBuffer = await readFile(filePath);
      
      return new NextResponse(fileBuffer, {
        headers: {
          'Content-Type': document.mimetype,
          'Content-Disposition': `inline; filename="${document.originalName}"`,
          'Content-Length': document.size.toString(),
        },
      });
    } catch (error) {
      console.error('Error reading file:', error);
      return NextResponse.json(
        { error: 'File not found on disk' },
        { status: 404 }
      );
    }

  } catch (error) {
    console.error('Get file error:', error);
    return NextResponse.json(
      { error: 'Failed to retrieve file' },
      { status: 500 }
    );
  }
}