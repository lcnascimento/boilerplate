import Link from "next/link";
import { useFormatter, useTranslations } from "next-intl";

import type { Note } from "@boilerplate/api/notes/schemas";

interface NoteListProps {
  /** Newest first, as the service lists them. */
  notes: Note[];
}

export function NoteList({ notes }: NoteListProps) {
  const t = useTranslations("notes");
  const format = useFormatter();

  return (
    <ul aria-label={t("list")} className="grid gap-3">
      {notes.map((note) => (
        <li key={note.id}>
          <Link
            className="block rounded-xl border p-4 transition-colors hover:bg-accent focus-visible:ring-[3px] focus-visible:ring-ring/50 focus-visible:outline-none"
            href={`/notes/${note.id}`}
          >
            <span className="block font-medium">{note.title}</span>
            {note.body ? (
              <span className="mt-1 line-clamp-2 block text-sm text-muted-foreground">
                {note.body}
              </span>
            ) : null}
            <time
              className="mt-2 block text-xs text-muted-foreground"
              dateTime={note.createdAt.toISOString()}
            >
              {format.dateTime(note.createdAt, {
                dateStyle: "medium",
                timeStyle: "short",
              })}
            </time>
          </Link>
        </li>
      ))}
    </ul>
  );
}
