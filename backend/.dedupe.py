import sys
path = 'models/types.go'
src = open(path, encoding='utf-8').read()
BT = chr(96)
dup = ('// ChangeEmailRequest binds or replaces the account email. Password' + chr(10)
       + '// confirmation is required.' + chr(10)
       + 'type ChangeEmailRequest struct {' + chr(10)
       + T + 'CurrentPassword string ' + BT + 'json:"current_password"' + BT + chr(10)
       + T + 'NewEmail        string ' + BT + 'json:"new_email"' + BT + chr(10)
       + '}')
pass