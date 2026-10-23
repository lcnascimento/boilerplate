# Boilerplate

Boilerplate de um sistema de anotações, usado como referência para a arquitetura de serviços, frontend e mensageria do Projeto.

## Language

### Access

**User**:
Uma pessoa que se autentica no sistema por email e senha.
_Avoid_: Usuário, conta, account

**Organization**:
O tenant: o espaço ao qual as Notes pertencem e no qual Users atuam como Members.
_Avoid_: Workspace, tenant, time

**Member**:
O vínculo de um User com uma Organization, com exatamente uma Role.
_Avoid_: Membership, participante

**Role**:
O papel de um Member na sua Organization, que define o que ele pode fazer: Owner, Admin ou Viewer.
_Avoid_: Perfil, permissão, grupo

**Permission**:
Uma ação sobre um tipo de recurso que uma Role concede ou não (ex.: apagar Note).
_Avoid_: Capability, privilégio, ability

**Owner**:
O único Member com todos os poderes na Organization, inclusive apagar Notes e transferir a propriedade a outro Member.
_Avoid_: Dono, superadmin

**Admin**:
Role que lê e cria Notes e gere Members, mas não apaga Notes nem transfere a propriedade.
_Avoid_: Editor, moderador

**Viewer**:
Role que só lê Notes.
_Avoid_: Leitor, read-only, member

**Actor**:
O User agindo numa Organization, com a Role que tem nela, numa dada requisição.
_Avoid_: Principal, caller, current user

### Notes

**Note**:
Uma anotação com título e corpo, pertencente a uma Organization, identificada por um id canônico e datada pela sua criação.
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
