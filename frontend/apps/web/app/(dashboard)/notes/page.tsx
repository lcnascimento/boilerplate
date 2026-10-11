import { NotebookPen } from "lucide-react";
import { getTranslations } from "next-intl/server";
import { Suspense } from "react";

import { Skeleton } from "@boilerplate/ui/components/skeleton";
import { EmptyState } from "@boilerplate/ui/primitives/empty-state";
import { PageHeader } from "@boilerplate/ui/primitives/page-header";

import { listNotes } from "@/actions/notes";

import { CreateNoteDialog, NoteList, NotesError } from "./_components";

export default async function NotesPage() {
  const t = await getTranslations("notes");

  return (
    <>
      <PageHeader
        action={<CreateNoteDialog />}
        description={t("description")}
        title={t("title")}
      />
      <Suspense fallback={<Skeleton aria-busy className="h-48 w-full" />}>
        <Notes />
      </Suspense>
    </>
  );
}

async function Notes() {
  const [t, result] = await Promise.all([
    getTranslations("notes"),
    listNotes(),
  ]);

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
