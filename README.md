=====
This is a music plugin for the bruxism multi-service bot.

This code is in very early stages and is still a bit messy.  


### Requirements

* [ffmpeg](http://ffmpeg.org/) must be installed and within your PATH
* [youtube-dl](https://github.com/rg3/youtube-dl) must be installed and exist in the same folder as bruxism.
* [dca](https://github.com/bwmarrin/dca) must be installed and exist in the same folder as bruxism.

### Commands

* NOTE: All commands must be prefixed with a @mention of your bot

| Command         | Arguments                        | Action
| --------------- | -------------------------------- | ------------------- 
| **mu**sic join  | [<channel ID>|<channel name>]    | Join the provided channel or if no channel is provided then join the last channel that was used.   


### Notes

For faster startup of youtube-dl use github cloned version of youtube-dl, see 
https://www.raspberrypi.org/forums/viewtopic.php?f=38&t=83763

To avoid youtube-dl ffmpeg post processing error, use a youtube-dl version past
this commit https://github.com/rg3/youtube-dl/commit/e38cafe986994d65230e6518752def8b53ad7308 
see issue https://github.com/rg3/youtube-dl/issues/8729#event-574790956
