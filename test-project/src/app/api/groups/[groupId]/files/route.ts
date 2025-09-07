import { NextRequest, NextResponse } from "next/server";
import { db } from "@/lib/db";
import { auth } from "@/lib/auth";
import { headers } from "next/headers";
import { writeFile, mkdir } from "fs/promises";
import path from "path";
import { existsSync } from "fs";

// POST /api/groups/[groupId]/files - Upload a file to the group
export async function POST(
  req: NextRequest,
  context: { params: Promise<{ groupId: string }> }
) {
  try {
    const { groupId } = await context.params;
    const session = await auth.api.getSession({
      headers: await headers(),
    });

    // Check if group exists and is active
    const group = await db.group.findUnique({
      where: { id: groupId },
      include: {
        members: true,
      },
    });

    if (!group) {
      return NextResponse.json(
        { error: "Group not found" },
        { status: 404 }
      );
    }

    if (!group.isActive || new Date(group.expiresAt) < new Date()) {
      return NextResponse.json(
        { error: "Group is expired or inactive" },
        { status: 403 }
      );
    }

    // For authenticated users, check if they're a member
    if (session?.user?.id) {
      const isMember = group.members.some(
        (m) => m.userId === session.user.id
      );
      if (!isMember) {
        return NextResponse.json(
          { error: "You must be a member of this group to upload files" },
          { status: 403 }
        );
      }
    }

    // Parse the form data
    const formData = await req.formData();
    const file = formData.get("file") as File;

    if (!file) {
      return NextResponse.json(
        { error: "No file provided" },
        { status: 400 }
      );
    }

    // Create the group-specific folder if it doesn't exist
    const uploadDir = path.join(
      process.cwd(),
      "public",
      "uploads",
      group.storageFolder
    );

    if (!existsSync(uploadDir)) {
      await mkdir(uploadDir, { recursive: true });
    }

    // Generate a unique filename
    const timestamp = Date.now();
    const filename = `${timestamp}-${file.name}`;
    const filepath = path.join(uploadDir, filename);
    const publicPath = `/uploads/${group.storageFolder}/${filename}`;

    // Convert the file to a buffer and save it
    const bytes = await file.arrayBuffer();
    const buffer = Buffer.from(bytes);
    await writeFile(filepath, buffer);

    // Save file metadata to database
    const groupFile = await db.groupFile.create({
      data: {
        filename,
        originalName: file.name,
        mimetype: file.type || "application/octet-stream",
        size: buffer.length,
        path: publicPath,
        uploaderId: session?.user?.id || null,
        groupId: group.id,
        isFromCreator: group.creatorId === session?.user?.id || false,
      },
      include: {
        uploader: {
          select: {
            id: true,
            name: true,
            email: true,
          },
        },
      },
    });

    return NextResponse.json(groupFile);
  } catch (error) {
    console.error("Failed to upload file:", error);
    return NextResponse.json(
      { error: "Failed to upload file" },
      { status: 500 }
    );
  }
}

// GET /api/groups/[groupId]/files - Get all files for a group
export async function GET(
  req: NextRequest,
  context: { params: Promise<{ groupId: string }> }
) {
  try {
    const { groupId } = await context.params;

    const files = await db.groupFile.findMany({
      where: { groupId },
      include: {
        uploader: {
          select: {
            id: true,
            name: true,
            email: true,
          },
        },
      },
      orderBy: {
        createdAt: "desc",
      },
    });

    return NextResponse.json(files);
  } catch (error) {
    console.error("Failed to fetch files:", error);
    return NextResponse.json(
      { error: "Failed to fetch files" },
      { status: 500 }
    );
  }
}