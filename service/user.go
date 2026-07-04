package service

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/gokins/gokins/comm"
	"github.com/gokins/gokins/model"
	"github.com/gokins/gokins/util"
	"github.com/sirupsen/logrus"
)

// GetUser fetches a user by ID using the global context.
// Prefer GetUserCtx when a request context is available.
func GetUser(uid string) (*model.TUser, bool) {
	return GetUserCtx(comm.Ctx, uid)
}

// GetUserCtx fetches a user by ID with the provided context for cancellation/timeout.
func GetUserCtx(ctx context.Context, uid string) (*model.TUser, bool) {
	if uid == "" {
		return nil, false
	}
	e := &model.TUser{}
	ok, err := comm.Db.Context(ctx).Where("id=?", uid).Get(e)
	if err != nil {
		logrus.Errorf("GetUser(%s) err:%v", uid, err)
	}
	return e, ok
}

// GetUserInfo fetches user info by ID using the global context.
// Prefer GetUserInfoCtx when a request context is available.
func GetUserInfo(uid string) (*model.TUserInfo, bool) {
	return GetUserInfoCtx(comm.Ctx, uid)
}

// GetUserInfoCtx fetches user info by ID with the provided context.
func GetUserInfoCtx(ctx context.Context, uid string) (*model.TUserInfo, bool) {
	if uid == "" {
		return nil, false
	}
	e := &model.TUserInfo{Id: uid}
	ok, err := comm.Db.Context(ctx).Where("id=?", uid).Get(e)
	if err != nil {
		logrus.Errorf("GetUser(%s) err:%v", uid, err)
	}
	return e, ok
}

// FindUserName looks up a user by name using the global context.
// Prefer FindUserNameCtx when a request context is available.
func FindUserName(name string) (*model.TUser, bool) {
	return FindUserNameCtx(comm.Ctx, name)
}

// FindUserNameCtx looks up a user by name with the provided context.
func FindUserNameCtx(ctx context.Context, name string) (*model.TUser, bool) {
	e := &model.TUser{}
	ok, err := comm.Db.Context(ctx).Where("name=?", name).Get(e)
	if err != nil {
		logrus.Errorf("FindUserName(%s) err:%v", name, err)
	}
	return e, ok
}

func ClearUserCache(uid string) {
	if uid == "" {
		return
	}
	uids := fmt.Sprintf("user:%s", uid)
	if err := comm.CacheSet(uids, nil); err != nil {
		logrus.Warnf("ClearUserCache: failed to clear cache for uid %s: %v", uid, err)
	}
}

// GetUserCache retrieves a user from cache or database using the global context.
// Prefer GetUserCacheCtx when a request context is available for cancellation/timeout support.
func GetUserCache(uid string) (*model.TUser, bool) {
	return GetUserCacheCtx(comm.Ctx, uid)
}

// GetUserCacheCtx retrieves a user from cache or database with context support.
// This enables request-scoped cancellation and timeout for database queries.
func GetUserCacheCtx(ctx context.Context, uid string) (*model.TUser, bool) {
	var ok bool
	e := &model.TUser{}
	uids := fmt.Sprintf("user:%s", uid)
	err := comm.CacheGets(uids, e)
	if err == nil {
		return e, true
	}
	e, ok = GetUserCtx(ctx, uid)
	if ok {
		if err := comm.CacheSets(uids, e); err != nil {
			logrus.Warnf("GetUserCacheCtx: failed to cache user %s: %v", uid, err)
		}
	}
	return e, ok
}
// CurrUserCache retrieves the current user from the request context.
// Prefer CurrUserCacheCtx when you need to pass a custom context.
func CurrUserCache(c *gin.Context) (*model.TUser, bool) {
	if c == nil {
		return nil, false
	}
	if c.Request == nil {
		return nil, false
	}
	return CurrUserCacheCtx(c.Request.Context(), c)
}

// CurrUserCacheCtx retrieves the current user using the provided context.
// This enables request-scoped cancellation and timeout for database queries.
func CurrUserCacheCtx(ctx context.Context, c *gin.Context) (*model.TUser, bool) {
	if c == nil {
		return nil, false
	}
	if c.Request == nil {
		return nil, false
	}
	tk := util.GetToken(c, comm.Cfg.Server.LoginKey)
	if tk == nil {
		return nil, false
	}
	uid, ok := tk["uid"]
	if !ok {
		return nil, false
	}
	uids, ok := uid.(string)
	if !ok || uids == "" {
		return nil, false
	}
	return GetUserCacheCtx(ctx, uids)
}
// IsAdmin checks if a user is the super-admin.
func IsAdmin(usr *model.TUser) bool {
	return usr.Id == "admin"
}

// IsOrgAdmin checks if a user is an admin of the specified organization.
// Uses the global context; prefer IsOrgAdminCtx when a request context is available.
func IsOrgAdmin(uid, orgId string) bool {
	return IsOrgAdminCtx(comm.Ctx, uid, orgId)
}

// IsOrgAdminCtx is the context-aware version of IsOrgAdmin.
func IsOrgAdminCtx(ctx context.Context, uid, orgId string) bool {
	usero, ok := GetUserOrgCtx(ctx, uid, orgId)
	if !ok {
		return false
	}
	return usero.PermAdm != 0
}

// GetUsePermRwr retrieves the read-write permission level for a user in an organization.
// Uses the global context; prefer GetUsePermRwrCtx when a request context is available.
func GetUsePermRwr(uid, orgId string) int {
	return GetUsePermRwrCtx(comm.Ctx, uid, orgId)
}

// GetUsePermRwrCtx is the context-aware version of GetUsePermRwr.
func GetUsePermRwrCtx(ctx context.Context, uid, orgId string) int {
	usero, ok := GetUserOrgCtx(ctx, uid, orgId)
	if !ok {
		return 0
	}
	return usero.PermRw
}

// HasOrgExec checks if a user has execution permission in the specified organization.
// Uses the global context; prefer HasOrgExecCtx when a request context is available.
func HasOrgExec(uid, orgId string) bool {
	return HasOrgExecCtx(comm.Ctx, uid, orgId)
}

// HasOrgExecCtx is the context-aware version of HasOrgExec.
func HasOrgExecCtx(ctx context.Context, uid, orgId string) bool {
	usero, ok := GetUserOrgCtx(ctx, uid, orgId)
	if !ok {
		return false
	}
	return usero.PermExec != 0
}
func GetUserOrg(uid, orgId string) (*model.TUserOrg, bool) {
	return GetUserOrgCtx(comm.Ctx, uid, orgId)
}

// GetUserOrgCtx is the context-aware version of GetUserOrg.
func GetUserOrgCtx(ctx context.Context, uid, orgId string) (*model.TUserOrg, bool) {
	torg := &model.TOrg{}
	ok := GetIdOrAidCtx(ctx, orgId, torg)
	if !ok {
		return nil, false
	}
	usero := &model.TUserOrg{}
	get, err := comm.Db.Context(ctx).Where("uid =? and org_id =?", uid, torg.Id).Get(usero)
	if err != nil {
		logrus.Debugf("HasOrgExec db err:%v", err)
	}
	if !get {
		return nil, false
	}
	return usero, true
}

type OrgPerm struct {
	lgusr  *model.TUser
	org    *model.TOrg
	usrOrg *model.TUserOrg
}

// NewOrgPerm creates an OrgPerm using the global context.
// Prefer NewOrgPermCtx when a request context is available.
func NewOrgPerm(lgusr *model.TUser, orgId string) *OrgPerm {
	return NewOrgPermCtx(comm.Ctx, lgusr, orgId)
}

// NewOrgPermCtx is the context-aware version of NewOrgPerm.
func NewOrgPermCtx(ctx context.Context, lgusr *model.TUser, orgId string) *OrgPerm {
	c := &OrgPerm{lgusr: lgusr}
	org := &model.TOrg{}
	ok := false
	if orgId != "" {
		ok = GetIdOrAidCtx(ctx, orgId, org)
	}
	if ok && org.Deleted != 1 {
		c.org = org
		usero := &model.TUserOrg{}
		if lgusr != nil {
			ok, err := comm.Db.Context(ctx).Where("uid =? and org_id =?", lgusr.Id, org.Id).Get(usero)
			if err != nil {
				logrus.Warnf("NewOrgPermCtx: failed to query user org (uid=%s, org=%s): %v", lgusr.Id, org.Id, err)
			}
			if ok {
				c.usrOrg = usero
			}
		}
	}
	return c
}
func (c *OrgPerm) IsAdmin() bool {
	if c.lgusr != nil && IsAdmin(c.lgusr) {
		return true
	}
	return false
}
func (c *OrgPerm) IsOrgOwner() bool {
	if c.org != nil && c.lgusr != nil && c.org.Uid == c.lgusr.Id {
		return true
	}
	return false
}
func (c *OrgPerm) IsOrgPublic() bool {
	if c.org != nil && c.org.Public == 1 {
		return true
	}
	return false
}
func (c *OrgPerm) IsOrgAdmin() bool {
	if c.IsAdmin() || c.IsOrgOwner() {
		return true
	}
	if c.usrOrg != nil && c.usrOrg.PermAdm == 1 {
		return true
	}
	return false
}
func (c *OrgPerm) CanRead() bool {
	if c.IsOrgPublic() || c.IsOrgAdmin() {
		return true
	}
	return c.usrOrg != nil
}
func (c *OrgPerm) CanWrite() bool {
	if c.IsOrgAdmin() {
		return true
	}
	if c.usrOrg != nil && c.usrOrg.PermRw == 1 {
		return true
	}
	return false
}
func (c *OrgPerm) CanDownload() bool {
	if c.IsOrgAdmin() {
		return true
	}
	if c.usrOrg != nil && c.usrOrg.PermDown == 1 {
		return true
	}
	return false
}
func (c *OrgPerm) CanExec() bool {
	if c.IsOrgAdmin() {
		return true
	}
	if c.usrOrg != nil && c.usrOrg.PermExec == 1 {
		return true
	}
	return false
}

// LgUser maybe null
func (c *OrgPerm) LgUser() *model.TUser {
	return c.lgusr
}

// Org maybe null
func (c *OrgPerm) Org() *model.TOrg {
	return c.org
}

// UserOrg maybe null
func (c *OrgPerm) UserOrg() *model.TUserOrg {
	return c.usrOrg
}

type UserPipeOrgPerm struct {
	OrgId     string `xorm:"org_id"`
	OrgName   string `xorm:"org_name"`
	OrgUid    string `xorm:"org_uid"`
	OrgPublic int    `xorm:"org_public"`
	OpPublic  int    `xorm:"op_public"`
	CurUid    string `xorm:"cur_uid"`
	PermAdm   int    `xorm:"perm_adm"`
	PermRw    int    `xorm:"perm_rw"`
	PermExec  int    `xorm:"perm_exec"`
}
type PipePerm struct {
	lgusr *model.TUser
	pipe  *model.TPipeline
	perms []*UserPipeOrgPerm
}

// NewPipePerm creates a PipePerm using the global context.
// Prefer NewPipePermCtx when a request context is available.
func NewPipePerm(lgusr *model.TUser, pipeId string) *PipePerm {
	return NewPipePermCtx(comm.Ctx, lgusr, pipeId)
}

// NewPipePermCtx is the context-aware version of NewPipePerm.
func NewPipePermCtx(ctx context.Context, lgusr *model.TUser, pipeId string) *PipePerm {
	c := &PipePerm{lgusr: lgusr}
	pipe := &model.TPipeline{}
	ok := false
	if pipeId != "" {
		var err error
		ok, err = comm.Db.Context(ctx).Where("id=?", pipeId).Get(pipe)
		if err != nil {
			logrus.Warnf("NewPipePermCtx: failed to query pipeline (id=%s): %v", pipeId, err)
		}
	}
	if ok {
		c.pipe = pipe
		if comm.IsMySQL && lgusr != nil {
			ses := comm.Db.Context(ctx).SQL(`
select org.id as org_id,org.name as org_name,org.uid as org_uid,org.public as org_public,op.public as op_public,
uo.uid as cur_uid,uo.perm_adm,uo.perm_rw,uo.perm_exec,uo.perm_down
from t_org org
JOIN t_org_pipe op ON op.pipe_id=? and org.id=op.org_id
LEFT JOIN t_user_org uo ON uo.uid=? and org.id=uo.org_id
where org.deleted!=1 or org.public=1
			`, pipe.Id, lgusr.Id)
			if err := ses.Find(&c.perms); err != nil {
				logrus.Warnf("NewPipePermCtx: failed to query org perms (pipe=%s, user=%s): %v", pipe.Id, lgusr.Id, err)
			}
		}
	}
	return c
}
func (c *PipePerm) IsAdmin() bool {
	if c.lgusr != nil && IsAdmin(c.lgusr) {
		return true
	}
	return false
}
func (c *PipePerm) IsPipeOwner() bool {
	if c.pipe != nil && c.lgusr != nil && c.pipe.Uid == c.lgusr.Id {
		return true
	}
	return false
}
func (c *PipePerm) CanRead() bool {
	if c.IsAdmin() || c.IsPipeOwner() {
		return true
	}
	for _, v := range c.perms {
		if c.lgusr != nil && v.OrgUid == c.lgusr.Id {
			return true
		}
		if v.OrgPublic == 1 {
			return true
		}
		if v.CurUid != "" {
			return true
		}
	}
	return false
}
func (c *PipePerm) CanWrite() bool {
	if c.IsAdmin() || c.IsPipeOwner() {
		return true
	}
	for _, v := range c.perms {
		if c.lgusr != nil && v.OrgUid == c.lgusr.Id {
			return true
		}
		if v.CurUid != "" && (v.PermAdm == 1 || v.PermRw == 1) {
			return true
		}
	}
	return false
}
func (c *PipePerm) CanExec() bool {
	if c.IsAdmin() || c.IsPipeOwner() {
		return true
	}
	for _, v := range c.perms {
		if c.lgusr != nil && v.OrgUid == c.lgusr.Id {
			return true
		}
		if v.CurUid != "" && (v.PermAdm == 1 || v.PermExec == 1) {
			return true
		}
	}
	return false
}

// LgUser maybe null
func (c *PipePerm) LgUser() *model.TUser {
	return c.lgusr
}

// Pipeline maybe null
func (c *PipePerm) Pipeline() *model.TPipeline {
	return c.pipe
}
