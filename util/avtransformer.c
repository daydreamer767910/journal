
#include "avtransformer.h"

int addSubtitlesToVideo(const char *input_video, const char **input_subtitles, int num_subtitles, const char *output_video) {
    AVFormatContext *input_format_ctx = NULL, *output_format_ctx = NULL;
    AVPacket packet;
    int ret, stream_index;

    // 打开输入视频文件
    ret = avformat_open_input(&input_format_ctx, input_video, NULL, NULL);
    if (ret < 0) {
        fprintf(stderr, "Error opening input video file: %s\n", av_err2str(ret));
        return ret;
    }
printf("1111111111111111111111111111111\n");
    // 获取视频流信息
    ret = avformat_find_stream_info(input_format_ctx, NULL);
    if (ret < 0) {
        fprintf(stderr, "Error finding stream information: %s\n", av_err2str(ret));
        avformat_close_input(&input_format_ctx);
        return ret;
    }
printf("22222222222222222222222222222222\n");
    // 打开输出视频文件
    ret = avformat_alloc_output_context2(&output_format_ctx, NULL, NULL, output_video);
    if (ret < 0) {
        fprintf(stderr, "Error allocating output context: %s\n", av_err2str(ret));
        avformat_close_input(&input_format_ctx);
        return ret;
    }
printf("333333333333333333333333333333333\n");
    // 遍历输入视频文件的流并复制到输出文件中
    for (stream_index = 0; stream_index < input_format_ctx->nb_streams; stream_index++) {
        AVStream *in_stream = input_format_ctx->streams[stream_index];
        AVStream *out_stream = avformat_new_stream(output_format_ctx, NULL);
        if (!out_stream) {
            fprintf(stderr, "Failed allocating output stream\n");
            ret = AVERROR_UNKNOWN;
            goto end;
        }
        ret = avcodec_parameters_copy(out_stream->codecpar, in_stream->codecpar);
        if (ret < 0) {
            fprintf(stderr, "Failed to copy codec parameters: %s\n", av_err2str(ret));
            goto end;
        }
        out_stream->codecpar->codec_tag = 0;
    }
printf("4444444444444444444444444444444444\n");
// 写入输出文件的文件头
    ret = avformat_write_header(output_format_ctx, NULL);
    if (ret < 0) {
        fprintf(stderr, "Error writing header: %s\n", av_err2str(ret));
        goto end;
    }
printf("55555555555555555555555555555555555\n");
    // 添加字幕流到输出视频文件
    for (int i = 0; i < num_subtitles; i++) {
        // 添加字幕流
        AVStream *subtitle_stream = avformat_new_stream(output_format_ctx, NULL);
        if (!subtitle_stream) {
            fprintf(stderr, "Failed allocating subtitle stream\n");
            ret = AVERROR_UNKNOWN;
            goto end;
        }

        // 设置字幕流的参数
        AVCodecParameters *codecpar = subtitle_stream->codecpar;
        codecpar->codec_type = AVMEDIA_TYPE_SUBTITLE;
        codecpar->codec_id = AV_CODEC_ID_TEXT;

        // 打开字幕文件
        FILE *subtitle_file = fopen(input_subtitles[i], "r");
        if (!subtitle_file) {
            fprintf(stderr, "Failed to open subtitle file: %s\n", input_subtitles[i]);
            ret = AVERROR_UNKNOWN;
            goto end;
        }

        // 读取并写入字幕数据
        transferSubtitles(subtitle_file,output_format_ctx,subtitle_stream);

        // 关闭字幕文件
        fclose(subtitle_file);
    }

printf("666666666666666666666666666666666666\n");
    // 写入流数据
    while (1) {
        ret = av_read_frame(input_format_ctx, &packet);
        if (ret < 0) break;

        AVStream *in_stream, *out_stream;
        in_stream = input_format_ctx->streams[packet.stream_index];
        out_stream = output_format_ctx->streams[packet.stream_index];

        // 调整包的时间戳等信息
        av_packet_rescale_ts(&packet, in_stream->time_base, out_stream->time_base);
        packet.pts = av_rescale_q_rnd(packet.pts, in_stream->time_base, out_stream->time_base, AV_ROUND_NEAR_INF | AV_ROUND_PASS_MINMAX);
        packet.dts = av_rescale_q_rnd(packet.dts, in_stream->time_base, out_stream->time_base, AV_ROUND_NEAR_INF | AV_ROUND_PASS_MINMAX);
        packet.duration = av_rescale_q(packet.duration, in_stream->time_base, out_stream->time_base);
        packet.pos = -1;

        // 写包到输出流
        av_interleaved_write_frame(output_format_ctx, &packet);
        av_packet_unref(&packet);
    }
printf("77777777777777777777777777777777777777\n");
    // 写输出文件的文件尾
    ret = av_write_trailer(output_format_ctx);
    if (ret < 0) {
        fprintf(stderr, "Error writing trailer: %s\n", av_err2str(ret));
    }

end:
    // 清理资源
    avformat_close_input(&input_format_ctx);
    if (output_format_ctx && !(output_format_ctx->oformat->flags & AVFMT_NOFILE))
        avio_closep(&output_format_ctx->pb);
    avformat_free_context(output_format_ctx);

    return ret;
}

int transferSubtitles(FILE *subtitle_file,AVFormatContext *output_format_ctx, AVStream *subtitle_stream) {
    int subtitle_index = 0;
    char line[1024];
    AVPacket packet;

    while (fgets(line, sizeof(line), subtitle_file)) {
        printf("1 line parsed\n");
        line[strcspn(line, "\n")] = '\0';
        // 解析字幕块
        if (sscanf(line, "%d", &subtitle_index) == 1) {
            printf("idx %d:[%s]\n",subtitle_index,line);
            // 时间轴格式为 hh:mm:ss,sss --> hh:mm:ss,sss
            char start_time_str[12], end_time_str[12];
            int start_hour, start_minute, start_second, start_millisecond;
            int end_hour, end_minute, end_second, end_millisecond;

            const char splitStr[3] = "-->";
            char *token = NULL;
            if (fgets(line, sizeof(line), subtitle_file)) {
                char buff[1024];
                printf("2 line parsed\n");
                line[strcspn(line, "\n")] = '\0';
                memcpy(buff,line,1024);
                token = strtok(buff,splitStr);
                if( token != NULL ) {
                    printf( "%s\n", token );
                    strncpy(start_time_str,token,12);
                    token = strtok(NULL, splitStr);
                }
                if( token != NULL ) {
                    printf( "%s\n", token );
                    strncpy(end_time_str,token,12);
                }

                sscanf(start_time_str, "%d:%d:%d,%d", &start_hour, &start_minute, &start_second, &start_millisecond);
                sscanf(end_time_str, "%d:%d:%d,%d", &end_hour, &end_minute, &end_second, &end_millisecond);
                printf("start:%s end:%s\n",start_time_str,end_time_str);
                int64_t start_time = start_hour * 3600000000LL + start_minute * 60000000LL + start_second * 1000000LL + start_millisecond * 1000LL;
                int64_t end_time = end_hour * 3600000000LL + end_minute * 60000000LL + end_second * 1000000LL + end_millisecond * 1000LL;
                        printf("start:%d end:%d\n",start_time,end_time);
                // 写入字幕数据包
                //AVStream *subtitle_stream = output_format_ctx->streams[packet.stream_index];
                packet.pts = av_rescale_q(start_time, (AVRational){1, AV_TIME_BASE}, subtitle_stream->time_base);
                packet.dts = av_rescale_q(end_time, (AVRational){1, AV_TIME_BASE}, subtitle_stream->time_base);
                packet.duration = av_rescale_q(end_time - start_time, (AVRational){1, AV_TIME_BASE}, subtitle_stream->time_base);

                int data_size = 0;
                char *ptext = NULL;
                packet.data = NULL;
                while (fgets(line, sizeof(line), subtitle_file) && strlen(line) > 1) {
                    line[strcspn(line, "\n")] = '\0';
                    data_size += strlen(line);
                    ptext = av_malloc(data_size);
                    if(!packet.data) {
                        strncpy(ptext,line,strlen(line));
                    } else {
                        strcpy(ptext,packet.data);
                        strcat(ptext,line);
                        av_free(packet.data);
                    }
                    packet.data = ptext;
                }
                printf("pts[%d]dts[%d]duration[%d]txt[%s]\n",
                    packet.pts,
                    packet.dts,
                    packet.duration,
                    packet.data);
                // 写包到输出流
                av_interleaved_write_frame(output_format_ctx, &packet);
                printf("free allocated resources\n");
                av_free(packet.data);
                av_packet_unref(&packet);
            }
            printf("one item parsed\n");
        }
        printf("one file parsed\n");
    }
    
    // 关闭字幕文件
    fclose(subtitle_file);
    
	printf("subtitles file parsed\n");
    return 0;
}
