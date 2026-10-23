import { getTranslations } from "next-intl/server";
import { Suspense } from "react";

import { Skeleton } from "@boilerplate/ui/components/skeleton";
import { PageHeader } from "@boilerplate/ui/primitives/page-header";

import { listNotes } from "@/actions/notes";
import { getRequestActor } from "@/shared/auth/request-actor";

import { canNote, CreateNoteDialog, Notes } from "./_components";

export default async function NotesPage() {
  const notes = listNotes();
  const [t, requestActor] = await Promise.all([
    getTranslations("notes"),
    getRequestActor(),
  ]);
  const role = requestActor.kind === "actor" ? requestActor.actor.role : null;
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
