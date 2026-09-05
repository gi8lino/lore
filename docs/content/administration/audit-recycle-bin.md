# Audit and recycle bin

## Audit log

Application services record administrative and content-management actions such as page deletion/moves/reviews, settings changes, snippet changes, imports, and other mutations. Administrators can inspect recent audit events with actor, action, object type/key, detail, and timestamp.

## Recycle bin

Deleting a page moves it into the recycle bin rather than immediately removing it permanently. An administrator can restore a deleted page or permanently delete it from the bin.

A deleted page continues to reserve its path. Creating a new page on the same path fails until the recycled page is restored/moved or permanently removed.
