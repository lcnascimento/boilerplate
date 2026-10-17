# Boilerplate

Boilerplate de um sistema de anotações, usado como referência para a arquitetura de serviços, frontend e mensageria do Projeto.

## Language

### Notes

**Note**:
Uma anotação com título e corpo, identificada por um id canônico e datada pela sua criação.
_Avoid_: Memo, post

### Events

**Integration Event**:
Fato consumado sobre o estado do sistema, publicado no event bus como contrato público e versionado para outros serviços consumirem.
_Avoid_: Domain event, mensagem, notificação

**NoteCreated**:
Integration Event que registra que uma Note passou a existir.
_Avoid_: NoteCreate, NoteAdded

**NoteDeleted**:
Integration Event que registra que uma Note deixou de existir.
_Avoid_: NoteDelete, NoteRemoved

### Audit

**Audit Entry**:
Registro imutável de um Integration Event recebido, guardado com seu envelope e payload sem interpretação.
_Avoid_: Log de evento, registro de histórico

**Audit Log**:
O conjunto de todas as Audit Entries; trilha de auditoria, não fonte de verdade do estado.
_Avoid_: Event store, histórico
