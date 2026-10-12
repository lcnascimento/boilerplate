import { NotebookPen } from "lucide-react";
import { useTranslations } from "next-intl";
import { use } from "react";

import { EmptyState } from "@boilerplate/ui/primitives/empty-state";

import { NoteList } from "./note-list";
import { NotesError } from "./notes-error";

import type { Result } from "@boilerplate/api/core";
import type { Note } from "@boilerplate/api/notes/schemas";

interface NotesProps {
  /** Started by the page, so the header streams before the list resolves. */
  notes: Promise<Result<Note[]>>;
}

/** The notes read: the list, its empty state or the error with a retry. */
export function Notes({ notes }: NotesProps) {
  const t = useTranslations("notes");
  const result = use(notes);

  if (result.kind === "error") {
    return <NotesError error={{ code: result.error.code }} />;
  }
  if (result.data.length === 0) {
    return (
      <EmptyState
        description={t("emptyDescription")}
        icon={NotebookPen}
        title={t("emptyTitle")}
      />
    );
  }
  return <NoteList notes={result.data} />;
}
