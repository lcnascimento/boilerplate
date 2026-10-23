import { ArrowLeft } from "lucide-react";
import Link from "next/link";
import { getFormatter, getTranslations } from "next-intl/server";

import { Button } from "@boilerplate/ui/components/button";
import { Card, CardContent } from "@boilerplate/ui/components/card";
import { PageHeader } from "@boilerplate/ui/primitives/page-header";

import { getNote } from "@/actions/notes";
import { getRequestActor } from "@/shared/auth/request-actor";
import { NOTES_HREF } from "@/shared/navigation";

import { canNote, DeleteNoteDialog, NotesError } from "../_components";

interface NotePageProps {
  params: Promise<{ id: string }>;
}

export default async function NotePage({ params }: NotePageProps) {
  const { id } = await params;
  const [t, format, result, requestActor] = await Promise.all([
    getTranslations("notes"),
    getFormatter(),
    getNote(id),
    getRequestActor(),
  ]);

  const role = requestActor.kind === "actor" ? requestActor.actor.role : null;

  const back = (
    <Button asChild className="self-start" variant="ghost">
      <Link href={NOTES_HREF}>
        <ArrowLeft aria-hidden />
        {t("backToList")}
      </Link>
    </Button>
  );

  if (result.kind === "error") {
    return (
      <>
        {back}
        <NotesError error={{ code: result.error.code }} />
      </>
    );
  }

  const note = result.data;
  return (
    <>
      {back}
      <PageHeader
        action={
          canNote(role, "delete") ? (
            <DeleteNoteDialog id={note.id} title={note.title} />
          ) : null
        }
        title={note.title}
        description={
          <time dateTime={note.createdAt.toISOString()}>
            {t("createdAt", {
              date: format.dateTime(note.createdAt, {
                dateStyle: "long",
                timeStyle: "short",
              }),
            })}
          </time>
        }
      />
      <Card>
        <CardContent>
          {note.body ? (
            <p className="whitespace-pre-wrap">{note.body}</p>
          ) : (
            <p className="text-muted-foreground">{t("noBody")}</p>
          )}
        </CardContent>
      </Card>
    </>
  );
}
