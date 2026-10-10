import { NotebookPen } from "lucide-react";
import { getTranslations } from "next-intl/server";

import { EmptyState } from "@boilerplate/ui/primitives/empty-state";
import { PageHeader } from "@boilerplate/ui/primitives/page-header";

import { listNotes } from "@/actions/notes";

import { CreateNoteDialog, NoteList, NotesError } from "./_components";

export default async function NotesPage() {
  const [t, result] = await Promise.all([
    getTranslations("notes"),
    listNotes(),
  ]);

  return (
    <>
      <PageHeader
        action={<CreateNoteDialog />}
        description={t("description")}
        title={t("title")}
      />
      {result.kind === "error" ? (
        <NotesError error={{ code: result.error.code }} />
      ) : result.data.length === 0 ? (
        <EmptyState
          description={t("emptyDescription")}
          icon={NotebookPen}
          title={t("emptyTitle")}
        />
      ) : (
        <NoteList notes={result.data} />
      )}
    </>
  );
}
