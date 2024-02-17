#ifndef AVTRANSFORMER_H
#define AVTRANSFORMER_H

#include <libavformat/avformat.h>
#include <libavcodec/avcodec.h>
#include <libavutil/error.h>
#include <libavutil/opt.h>
#include <libavutil/dict.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

int addSubtitlesToVideo(const char *input_video, const char **input_subtitles, int num_subtitles, const char *output_video);
int transferSubtitles(FILE *subtitle_file,AVFormatContext *output_format_ctx, AVStream *subtitle_stream);

#endif
