package convo

func (u *User) GetOID() string {
	return u.StringID()
}

func (u *User) GetUID() string {
	return u.Username
}

func (u *User) GetName() string {
	return u.Nickname
}

func (u *User) GetAvatar() string {
	return u.AvatarPath
}
