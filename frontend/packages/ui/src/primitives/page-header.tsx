import { cn } from "cn";

import type * as React from "react";

interface PageHeaderProps extends Omit<
  React.ComponentProps<"header">,
  "title"
> {
  title: React.ReactNode;
  description?: React.ReactNode;
  /** Page-level action, such as a "New" button, aligned to the end. */
  action?: React.ReactNode;
}

function PageHeader({
  className,
  title,
  description,
  action,
  ...props
}: PageHeaderProps) {
  return (
    <header
      data-slot="page-header"
      className={cn(
        "flex flex-col gap-4 pb-6 md:flex-row md:items-center md:justify-between",
        className
      )}
      {...props}
    >
      <div className="flex flex-col gap-1">
        <h1 className="text-2xl font-bold tracking-tight md:text-3xl">
          {title}
        </h1>
        {description ? (
          <p className="text-muted-foreground">{description}</p>
        ) : null}
      </div>
      {action ? <div className="shrink-0">{action}</div> : null}
    </header>
  );
}

export { PageHeader };
export type { PageHeaderProps };
