import landing from './landing'
import common from './common'
import dashboard from './dashboard'
import channelMonitorV2 from './channelMonitorV2'
import batchImage from './batchImage'
import lottery from './lottery'
import ranking from './ranking'
import imagePlayground from './imagePlayground'
import admin from './admin'
import misc from './misc'

export default {
  ...landing,
  ...common,
  ...dashboard,
  ...channelMonitorV2,
  ...batchImage,
  ...lottery,
  ...ranking,
  ...imagePlayground,
  admin,
  ...misc,
}
