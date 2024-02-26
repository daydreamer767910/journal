#ifndef AVTRANSFORMER_H
#define AVTRANSFORMER_H

#include <libavformat/avformat.h>
#include <libavcodec/avcodec.h>
#include <libavutil/error.h>
#include <libavutil/opt.h>
#include <libavutil/dict.h>
#include <libavutil/pixfmt.h>
#include <libavfilter/avfilter.h>
#include <libavfilter/buffersink.h>
#include <libavfilter/buffersrc.h>
#include <stdio.h>
#include <stdbool.h>
#include <stdlib.h>
#include <string.h>

int transferSubtitles(FILE *subtitle_file,AVFormatContext *output_format_ctx, AVStream *subtitle_stream);
int trans_vpacket(AVFormatContext *ic,AVCodecContext *dec_ctx,AVFormatContext *oc,AVCodecContext *en_ctx,AVFilterContext *src_ctx,AVFilterContext *sink_ctx);

/**
     * Merge the input video files into the ouput file.
     *
     * @param input_files    a video type file url list
     * @param num_input_files    the number of the video files
     * @param output_file    the output file usr
     * @param filters_str  the filters descrition list, This must be
     *                     in the 'name=options[inst_name],' form
	 *                     and ':'-separated list of options 
     *
     * @returns >=0 on success otherwise an error code.
     *          AVERROR(ENOSYS) on unsupported commands
     */
int mergeVideos(const char **input_files, int num_input_files, const char *output_file, const char *filters_str);
#endif
