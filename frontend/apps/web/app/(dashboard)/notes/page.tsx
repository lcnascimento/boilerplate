import { getTranslations } from "next-intl/server";
import { Suspense } from "react";

import { Skeleton } from "@boilerplate/ui/components/skeleton";
import { PageHeader } from "@boilerplate/ui/primitives/page-header";

import { getActiveRole } from "@/actions/members";
import { listNotes } from "@/actions/notes";

import { canNote, CreateNoteDialog, Notes } from "./_components";

export default async function NotesPage() {
  const notes = listNotes();
  const [t, role] = await Promise.all([
    getTranslations("notes"),
    getActiveRole(),
  ]);
  const canCreate = canNote(role, "create");

  return (
    <>
      <PageHeader
        action={canCreate ? <CreateNoteDialog /> : null}
        description={t("description")}
        title={t("title")}
      />
      <Suspense fallback={<Skeleton aria-busy className="h-48 w-full" />}>
        <Notes canCreate={canCreate} notes={notes} />
      </Suspense>
    </>
  );
}
