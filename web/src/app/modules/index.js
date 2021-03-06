/*
Copyright 2015 Gravitational, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

import reactor from 'app/reactor';

reactor.registerStores({
  'tlpt': require('./app/appStore'),  
  'tlpt_current_session': require('./currentSession/currentSessionStore'),
  'tlpt_user': require('./user/userStore'),
  'tlpt_sites': require('./sites/siteStore'),
  'tlpt_user_invite': require('./user/userInviteStore'),
  'tlpt_nodes': require('./nodes/nodeStore'),
  'tlpt_rest_api': require('./restApi/restApiStore'),
  'tlpt_sessions': require('./sessions/sessionStore'),
  'tlpt_stored_sessions_filter': require('./storedSessionsFilter/storedSessionFilterStore'),
  'tlpt_notifications': require('./notifications/notificationStore')
});
