import { getTranslations } from "next-intl/server";
import { Suspense } from "react";

import { Skeleton } from "@boilerplate/ui/components/skeleton";
import { PageHeader } from "@boilerplate/ui/primitives/page-header";

import { listNotes } from "@/actions/notes";

import { CreateNoteDialog, Notes } from "./_components";

export default async function NotesPage() {
  const notes = listNotes();
  const t = await getTranslations("notes");

  return (
    <>
      <PageHeader
        action={<CreateNoteDialog />}
        description={t("description")}
        title={t("title")}
      />
      <Suspense fallback={<Skeleton aria-busy className="h-48 w-full" />}>
        <Notes notes={notes} />
      </Suspense>
    </>
  );
}
