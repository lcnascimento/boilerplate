# Boilerplate

A linguagem de domínio do Projeto: cada termo tem um só sentido no código, na documentação e nas conversas.

## Language

### Access

**User**:
Uma pessoa que se autentica no sistema por email e senha.
_Avoid_: Usuário, conta, account

**Organization**:
O tenant: o espaço ao qual os dados do produto pertencem e no qual Users atuam como Members.
_Avoid_: Workspace, tenant, time

**Member**:
O vínculo de um User com uma Organization, com exatamente uma Role.
_Avoid_: Membership, participante

**Invitation**:
O convite, feito por um Owner ou Admin a um email, para que o User com esse email se torne Member da Organization com uma Role (Admin ou Viewer). Vale 7 dias e só vira Member se o convidado aceitar.
_Avoid_: Invite, adicionar Member

**Role**:
O papel de um Member na sua Organization, que define o que ele pode fazer: Owner, Admin ou Viewer.
_Avoid_: Perfil, permissão, grupo

**Permission**:
Uma ação sobre um tipo de recurso que uma Role concede ou não (ex.: transferir a propriedade da Organization).
_Avoid_: Capability, privilégio

**Management Permission**:
Uma Permission sobre a própria Organization, seus Members ou suas Invitations (ex.: trocar a Role de um Member). Nunca é condicionada a uma Feature Flag.
_Avoid_: Permission de membership, permissão administrativa

**Owner**:
O único Member com todos os poderes na Organization, inclusive excluí-la e transferir a propriedade a outro Member.
_Avoid_: Dono, superadmin

**Admin**:
Role que renomeia a Organization e gere Members e Invitations, mas não exclui a Organization nem transfere a propriedade.
_Avoid_: Editor, moderador

**Viewer**:
Role que só lê os dados da Organization, sem gerir Members nem Invitations.
_Avoid_: Leitor, read-only, member

**Actor**:
O User agindo numa Organization, com a Role que tem nela, numa dada requisição.
_Avoid_: Principal, caller, current user

**Feature Flag**:
Um interruptor que liga ou desliga uma funcionalidade para um User ou uma Organization, sem novo deploy.
_Avoid_: Toggle, feature switch

**Ability**:
O que um Actor pode fazer numa requisição: as Permissions da sua Role, exceto as condicionadas a uma Feature Flag desligada, mais as Feature Flags ligadas para ele.
_Avoid_: Access, entitlement, grant

<!-- example:begin -->

### Notes

**Note**:
Uma anotação com título e corpo, pertencente a uma Organization, identificada por um id canônico e datada pela sua criação. O Owner lê, cria e apaga Notes, o Admin lê e cria, e o Viewer só lê.
_Avoid_: Memo, post

<!-- example:end -->

### Events

**Integration Event**:
Fato consumado sobre o estado do sistema, publicado no event bus como contrato público e versionado para outros serviços consumirem.
_Avoid_: Domain event, mensagem, notificação

<!-- example:begin -->

**NoteCreated**:
Integration Event que registra que uma Note passou a existir.
_Avoid_: NoteCreate, NoteAdded

**NoteDeleted**:
Integration Event que registra que uma Note deixou de existir.
_Avoid_: NoteDelete, NoteRemoved

<!-- example:end -->

### Audit

**Audit Entry**:
Registro imutável de um Integration Event recebido, guardado com seu envelope e payload sem interpretação.
_Avoid_: Log de evento, registro de histórico

**Audit Log**:
O conjunto de todas as Audit Entries; trilha de auditoria, não fonte de verdade do estado.
_Avoid_: Event store, histórico
