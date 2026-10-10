import { Skeleton } from "@boilerplate/ui/components/skeleton";

/** Placeholder for a page: a header and a block of content. */
export function ContentSkeleton() {
  return (
    <div aria-busy className="flex flex-col gap-6">
      <div className="flex flex-col gap-2">
        <Skeleton className="h-8 w-64" />
        <Skeleton className="h-4 w-96 max-w-full" />
      </div>
      <Skeleton className="h-48 w-full" />
    </div>
  );
}
