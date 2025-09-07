-- RedefineTables
PRAGMA defer_foreign_keys=ON;
PRAGMA foreign_keys=OFF;
CREATE TABLE "new_GroupFile" (
    "id" TEXT NOT NULL PRIMARY KEY,
    "filename" TEXT NOT NULL,
    "originalName" TEXT NOT NULL,
    "mimetype" TEXT NOT NULL,
    "size" INTEGER NOT NULL,
    "path" TEXT NOT NULL,
    "uploaderId" TEXT,
    "groupId" TEXT NOT NULL,
    "isFromCreator" BOOLEAN NOT NULL,
    "thumbnailPath" TEXT,
    "createdAt" DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" DATETIME NOT NULL,
    CONSTRAINT "GroupFile_uploaderId_fkey" FOREIGN KEY ("uploaderId") REFERENCES "User" ("id") ON DELETE SET NULL ON UPDATE CASCADE,
    CONSTRAINT "GroupFile_groupId_fkey" FOREIGN KEY ("groupId") REFERENCES "Group" ("id") ON DELETE CASCADE ON UPDATE CASCADE
);
INSERT INTO "new_GroupFile" ("createdAt", "filename", "groupId", "id", "isFromCreator", "mimetype", "originalName", "path", "size", "thumbnailPath", "updatedAt", "uploaderId") SELECT "createdAt", "filename", "groupId", "id", "isFromCreator", "mimetype", "originalName", "path", "size", "thumbnailPath", "updatedAt", "uploaderId" FROM "GroupFile";
DROP TABLE "GroupFile";
ALTER TABLE "new_GroupFile" RENAME TO "GroupFile";
CREATE INDEX "GroupFile_groupId_idx" ON "GroupFile"("groupId");
CREATE INDEX "GroupFile_uploaderId_idx" ON "GroupFile"("uploaderId");
PRAGMA foreign_keys=ON;
PRAGMA defer_foreign_keys=OFF;
