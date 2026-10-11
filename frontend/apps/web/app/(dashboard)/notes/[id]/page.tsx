import { ArrowLeft } from "lucide-react";
import Link from "next/link";
import { getFormatter, getTranslations } from "next-intl/server";

import { Button } from "@boilerplate/ui/components/button";
import { Card, CardContent } from "@boilerplate/ui/components/card";
import { PageHeader } from "@boilerplate/ui/primitives/page-header";

import { getNote } from "@/actions/notes";
import { NOTES_HREF } from "@/shared/navigation";

import { DeleteNoteDialog, NotesError } from "../_components";

interface NotePageProps {
  params: Promise<{ id: string }>;
}

export default async function NotePage({ params }: NotePageProps) {
  const { id } = await params;
  const [t, format, result] = await Promise.all([
    getTranslations("notes"),
    getFormatter(),
    getNote(id),
  ]);

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
        action={<DeleteNoteDialog id={note.id} title={note.title} />}
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
