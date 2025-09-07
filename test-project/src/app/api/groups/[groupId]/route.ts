import { NextRequest, NextResponse } from "next/server";
import { db } from "@/lib/db";
import { auth } from "@/lib/auth";
import { headers } from "next/headers";

// GET /api/groups/[groupId] - Get a specific group
export async function GET(
  req: NextRequest,
  { params }: { params: Promise<{ groupId: string }> }
) {
  try {
    const { groupId } = await params;
    const group = await db.group.findUnique({
      where: {
        id: groupId,
      },
      include: {
        organization: true,
        members: {
          include: {
            user: {
              select: {
                id: true,
                name: true,
                email: true,
                image: true,
              },
            },
          },
        },
        files: true,
      },
    });

    if (!group) {
      return NextResponse.json(
        { error: "Group not found" },
        { status: 404 }
      );
    }

    // Check if group is expired
    const isExpired = new Date(group.expiresAt) < new Date();
    
    return NextResponse.json({
      ...group,
      isExpired,
      code: group.id.slice(0, 6).toUpperCase(),
    });
  } catch (error) {
    console.error("Failed to fetch group:", error);
    return NextResponse.json(
      { error: "Failed to fetch group" },
      { status: 500 }
    );
  }
}

// PATCH /api/groups/[groupId] - Update group (extend expiry)
export async function PATCH(
  req: NextRequest,
  { params }: { params: Promise<{ groupId: string }> }
) {
  try {
    const { groupId } = await params;
    const session = await auth.api.getSession({
      headers: await headers(),
    });

    const body = await req.json();
    const { action } = body;

    if (action === "extend") {
      // Check if user is the creator
      const membership = await db.groupMember.findFirst({
        where: {
          groupId: groupId,
          userId: session?.user?.id || "",
          role: "creator",
        },
      });

      if (!membership) {
        return NextResponse.json(
          { error: "Only the creator can extend the group" },
          { status: 403 }
        );
      }

      // Get current group
      const group = await db.group.findUnique({
        where: { id: groupId },
      });

      if (!group) {
        return NextResponse.json(
          { error: "Group not found" },
          { status: 404 }
        );
      }

      // Check if max extensions reached
      if (group.extendedCount >= 3) {
        return NextResponse.json(
          { error: "Maximum extensions reached" },
          { status: 400 }
        );
      }

      // Extend by 4 hours
      const newExpiry = new Date(group.expiresAt);
      newExpiry.setHours(newExpiry.getHours() + 4);

      // Update group
      const updatedGroup = await db.group.update({
        where: { id: groupId },
        data: {
          expiresAt: newExpiry,
          extendedCount: group.extendedCount + 1,
        },
      });

      return NextResponse.json(updatedGroup);
    }

    return NextResponse.json(
      { error: "Invalid action" },
      { status: 400 }
    );
  } catch (error) {
    console.error("Failed to update group:", error);
    return NextResponse.json(
      { error: "Failed to update group" },
      { status: 500 }
    );
  }
}

// DELETE /api/groups/[groupId] - Delete/Archive group
export async function DELETE(
  req: NextRequest,
  { params }: { params: Promise<{ groupId: string }> }
) {
  try {
    const { groupId } = await params;
    const session = await auth.api.getSession({
      headers: await headers(),
    });

    if (!session?.user?.id) {
      return NextResponse.json(
        { error: "Authentication required" },
        { status: 401 }
      );
    }

    // Check if user is the creator
    const membership = await db.groupMember.findFirst({
      where: {
        groupId: groupId,
        userId: session.user.id,
        role: "creator",
      },
    });

    if (!membership) {
      return NextResponse.json(
        { error: "Only the creator can delete the group" },
        { status: 403 }
      );
    }

    // Archive the group instead of deleting
    await db.group.update({
      where: { id: groupId },
      data: {
        isActive: false,
      },
    });

    return NextResponse.json({ success: true });
  } catch (error) {
    console.error("Failed to delete group:", error);
    return NextResponse.json(
      { error: "Failed to delete group" },
      { status: 500 }
    );
  }
}