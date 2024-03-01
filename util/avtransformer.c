#include <unistd.h>
#include "avtransformer.h"

static void display_frame(const AVFrame *frame, AVRational time_base)
{
    int x, y;
    uint8_t *p0, *p;
    int64_t delay;
    static int64_t last_pts = AV_NOPTS_VALUE;

    if (frame->pts != AV_NOPTS_VALUE) {
        if (last_pts != AV_NOPTS_VALUE) {
            /* sleep roughly the right amount of time;
             * usleep is in microseconds, just like AV_TIME_BASE. */
            delay = av_rescale_q(frame->pts - last_pts,
                                 time_base, AV_TIME_BASE_Q);
            if (delay > 0 && delay < 1000000)
                usleep(delay);
        }
        last_pts = frame->pts;
    }

    /* Trivial ASCII grayscale display. */
    p0 = frame->data[0];
    puts("\033c");
    for (y = 0; y < frame->height; y++) {
        p = p0;
        for (x = 0; x < frame->width; x++)
            putchar(" .-+#"[*(p++) / 52]);
        putchar('\n');
        p0 += frame->linesize[0];
    }
    fflush(stdout);
}

void print_stream_info(AVFormatContext *fmt_ctx,int out) {
    /*printf("File Information:\n");
    printf("  Duration: %lld milliseconds\n", fmt_ctx->duration / 1000);

    printf("Streams:\n");
    for (int i = 0; i < fmt_ctx->nb_streams; i++) {
        AVStream *stream = fmt_ctx->streams[i];
        const char *type_str = av_get_media_type_string(stream->codecpar->codec_type);
        printf("  Stream #%d: %s\n", i, type_str);

        printf("    Codec: %s\n", avcodec_get_name(stream->codecpar->codec_id));

        printf("    Duration: %lld milliseconds\n", stream->duration / AV_TIME_BASE * 1000);
        printf("    Time Base: %d/%d\n", stream->time_base.num, stream->time_base.den);

        if (stream->codecpar->codec_type == AVMEDIA_TYPE_VIDEO) {
            printf("    Frame Rate: %d/%d\n", stream->avg_frame_rate.num, stream->avg_frame_rate.den);
            printf("    Resolution: %dx%d\n", stream->codecpar->width, stream->codecpar->height);
        }
    }*/
    av_dump_format(fmt_ctx, 0, fmt_ctx->url, out);

}

static int select_channel_layout(const AVCodec *codec, AVChannelLayout *dst)
{
    const AVChannelLayout *p, *best_ch_layout;
    int best_nb_channels   = 0;

    if (!codec->ch_layouts)
        return av_channel_layout_copy(dst, &(AVChannelLayout)AV_CHANNEL_LAYOUT_STEREO);

    p = codec->ch_layouts;
    while (p->nb_channels) {
        int nb_channels = p->nb_channels;

        if (nb_channels > best_nb_channels) {
            best_ch_layout   = p;
            best_nb_channels = nb_channels;
        }
        p++;
    }
    return av_channel_layout_copy(dst, best_ch_layout);
}

int get_stream_index(AVFormatContext *format_ctx, enum AVMediaType etype) {
    for (int i = 0; i < format_ctx->nb_streams; i++) {
        if(format_ctx->streams[i]->codecpar->codec_type == etype)
            return i;
    }
    return -1;
}

AVFilterContext * createFilter(AVFilterGraph *filter_graph,char *name,char *instance_name, char *args) {
    AVFilterContext *filter_ctx = NULL;
    const AVFilter *filter = avfilter_get_by_name(name);
    if (!filter) {
        fprintf(stderr, "filtering %s element not found\n",name);
        return NULL;
    }
    if (avfilter_graph_create_filter(&filter_ctx, filter, instance_name, args, NULL, filter_graph) < 0) {
        fprintf(stderr, "Cannot create filter[%s] with args[%s]\n",instance_name,args);
        return NULL;
    }
    //printf("filter[%s][%s][%s]created\n",name,instance_name,args);
    return filter_ctx;
}

int create_vfilters_map(AVFilterGraph *graph,int num_ic ,AVCodecContext **dec_vctx_array,const char *filter_desc_str) {
    char all_filter_str[1024+strlen(filter_desc_str)];
    int ret = -1;
    memset(all_filter_str,0,sizeof(all_filter_str));
    for(int i=0; i<num_ic ;i++) {
        snprintf(all_filter_str, sizeof(all_filter_str), "%sbuffer=video_size=%dx%d:pix_fmt=%d:frame_rate=%d/%d:time_base=%d/%d:pixel_aspect=%d/%d[%d];",
            all_filter_str,
            dec_vctx_array[i]->width, 
            dec_vctx_array[i]->height, 
            dec_vctx_array[i]->pix_fmt,
            dec_vctx_array[i]->framerate.num,
            dec_vctx_array[i]->framerate.den,
            dec_vctx_array[i]->time_base.num, 
            dec_vctx_array[i]->time_base.den,
            dec_vctx_array[i]->sample_aspect_ratio.num, 
            dec_vctx_array[i]->sample_aspect_ratio.den,
            i);
    }
    snprintf(all_filter_str,sizeof(all_filter_str),"%s%s;[outv]buffersink",all_filter_str,filter_desc_str);

    printf("[%s]\n",all_filter_str);
    ret = avfilter_graph_parse_ptr(graph,all_filter_str,NULL,NULL,NULL);
    if (ret < 0) {
        fprintf(stderr, "Cannot parse filter map[%s] err[%s]\n",filter_desc_str,av_err2str(ret));
        return ret;
    }
    
    ret = avfilter_graph_config(graph, NULL);
    if (ret < 0) {
        fprintf(stderr, "Cannot config filter map[%s] err[%s]\n",filter_desc_str,av_err2str(ret));
        return ret;
    }
    return ret;
}


// 创建视频流的 AVCodecParameters
AVCodecParameters* create_video_codec_params(int width, int height, int bitrate, AVRational framerate, enum AVPixelFormat pix_fmt,AVRational aspect_ratio) {
    AVCodecParameters *codec_params = avcodec_parameters_alloc();
    if (!codec_params) {
        return NULL;
    }

    codec_params->codec_type = AVMEDIA_TYPE_VIDEO;
    codec_params->codec_id = AV_CODEC_ID_H264; // 例如：H.264编码器
    codec_params->width = width;
    codec_params->height = height;
    codec_params->bit_rate = bitrate;
    codec_params->format = pix_fmt;
    codec_params->framerate = framerate;
    codec_params->sample_aspect_ratio = aspect_ratio;
    
    // 其他视频流参数设置...

    return codec_params;
}


// 创建音频流的 AVCodecParameters
AVCodecParameters* create_audio_codec_params(int sample_rate, enum AVCodecID codec_id, int bitrate) {
    AVCodecParameters *codec_params = avcodec_parameters_alloc();
    if (!codec_params) {
        return NULL;
    }
    codec_params->codec_type = AVMEDIA_TYPE_AUDIO;
    codec_params->codec_id = codec_id; 
    codec_params->sample_rate = sample_rate;
    codec_params->bit_rate = bitrate;
    //codec_params->frame_size
    const AVCodec *codec = avcodec_find_encoder(codec_id);
    if (!codec) {
        fprintf(stderr, "Codec not found\n");
        avcodec_parameters_free(&codec_params);
        return NULL;
    }
    select_channel_layout(codec,&codec_params->ch_layout);
    
    // 其他音频流参数设置...

    return codec_params;
}

// 创建字幕流的 AVCodecParameters
AVCodecParameters* create_subtitle_codec_params() {
    AVCodecParameters *codec_params = avcodec_parameters_alloc();
    if (!codec_params) {
        return NULL;
    }

    codec_params->codec_type = AVMEDIA_TYPE_SUBTITLE;
    codec_params->codec_id = AV_CODEC_ID_TEXT; // 例如：文本字幕编码器
    //codec_params->format = pix_fmt;

    // 其他字幕流参数设置...
    
    return codec_params;
}
AVStream *create_stream(AVFormatContext *format_context, AVCodecParameters *codec_params,AVRational time_base) {
    AVStream *stream = avformat_new_stream(format_context, NULL);
    if (!stream) {
        fprintf(stderr, "Failed to allocate new stream\n");
        return NULL;
    }
    if (avcodec_parameters_copy(stream->codecpar, codec_params) < 0) {
        fprintf(stderr, "Failed to copy codec parameters to stream\n");
        return NULL;
    }
    stream->time_base = time_base;
    
    return stream;
}

AVCodecContext *create_codec_context(AVCodecParameters *codecpar,AVRational time_base, bool encode) {
    const AVCodec *codec = NULL;
    AVCodecContext *codec_ctx = NULL;

    // 查找解码器（或编码器）
    if(encode) {
        codec = avcodec_find_encoder(codecpar->codec_id);
    } else {
        codec = avcodec_find_decoder(codecpar->codec_id);
    }
    if (!codec) {
        // 如果找不到解码器（或编码器），处理错误
        fprintf(stderr, "Unsupported codec %d!\n",codecpar->codec_id);
        return NULL;
    }

    // 分配 AVCodecContext
    codec_ctx = avcodec_alloc_context3(codec);
    if (!codec_ctx) {
        fprintf(stderr, "Failed to allocate codec context!\n");
        return NULL;
    }

    // 将 AVCodecParameters 中的参数复制到 AVCodecContext 中
    if (avcodec_parameters_to_context(codec_ctx, codecpar) < 0) {
        fprintf(stderr, "Failed to copy codec parameters to codec context!\n");
        avcodec_free_context(&codec_ctx);
        return NULL;
    }
    codec_ctx->time_base = time_base;
    if(!encode)
        codec_ctx->pkt_timebase = time_base;
    
    // Open the codec
    int ret = avcodec_open2(codec_ctx, codec, NULL);
    if (ret < 0) {
        fprintf(stderr, "Error opening codec: %s\n", av_err2str(ret));
        avcodec_free_context(&codec_ctx);
        return NULL;
    }
    if(encode)
        printf("encode ");
    else
        printf("decode ");
    if(codec_ctx->codec_type == AVMEDIA_TYPE_VIDEO) {
        printf("video ");
        printf("codec ctx created:id[%d]w[%d]h[%d] timebase{%d,%d} aspect_rati{%d,%d} pix_fmt[%d] bit_rate[%d]framerate[%d,%d]\n",
            codec_ctx->codec_id,
            codec_ctx->width,
            codec_ctx->height,
            codec_ctx->time_base.den,
            codec_ctx->time_base.num,
            codec_ctx->sample_aspect_ratio.den,
            codec_ctx->sample_aspect_ratio.num,
            codec_ctx->pix_fmt,
            codec_ctx->bit_rate,
            codec_ctx->framerate.den,
            codec_ctx->framerate.num);
    }
    else if(codec_ctx->codec_type == AVMEDIA_TYPE_AUDIO) {
        printf("audio ");
        printf("codec ctx created:id[%d]timebase{%d,%d}pix_fmt[%d] bit_rate[%d] sample_rate[%d] sample_fmt[%d] channels[%d]\n",
            codec_ctx->codec_id,
            codec_ctx->time_base.den,
            codec_ctx->time_base.num,
            codec_ctx->pix_fmt,
            codec_ctx->bit_rate,
            codec_ctx->sample_rate,
            codec_ctx->sample_fmt,
            codec_ctx->ch_layout.nb_channels);
    }
    else {
        printf("type[%d] ",codec_ctx->codec_type);
        printf("codec ctx created:id[%d]w[%d]h[%d] timebase{%d,%d} aspect_rati{%d,%d} pix_fmt[%d] bit_rate[%d] sample_rate[%d] sample_fmt[%d] channels[%d]\n",
            codec_ctx->codec_id,
            codec_ctx->width,
            codec_ctx->height,
            codec_ctx->time_base.den,
            codec_ctx->time_base.num,
            codec_ctx->sample_aspect_ratio.den,
            codec_ctx->sample_aspect_ratio.num,
            codec_ctx->pix_fmt,
            codec_ctx->bit_rate,
            codec_ctx->sample_rate,
            codec_ctx->sample_fmt,
            codec_ctx->ch_layout.nb_channels);
    }
    return codec_ctx;
}


AVFormatContext * create_oc(const char *outputFileName,AVRational time_base,AVCodecParameters *v_params,AVCodecParameters *a_params) {
    AVFormatContext *formatCtx = NULL;
    int ret;

    // Open output file
    ret = avformat_alloc_output_context2(&formatCtx, NULL, NULL, outputFileName);
    if (ret < 0) {
        fprintf(stderr, "Error allocating output context: %s\n", av_err2str(ret));
        return NULL;
    }

    // Open output file
    if (!(formatCtx->oformat->flags & AVFMT_NOFILE)) {
        ret = avio_open(&formatCtx->pb, outputFileName, AVIO_FLAG_WRITE);
        if (ret < 0) {
            fprintf(stderr, "Error opening output file: %s\n", av_err2str(ret));
            goto fail;
        }
    }

    // Create a new video stream
    if(v_params) {
        if (!create_stream(formatCtx, v_params,time_base)) {
            fprintf(stderr, "Failed creating video stream\n");
            goto fail;
        }
    }
    // Create a new audio stream
    if(a_params) {
        if (!create_stream(formatCtx, a_params,time_base)) {
            fprintf(stderr, "Failed creating audio stream\n");
            goto fail;
        }
    }    
    return formatCtx;

fail:
    avformat_free_context(formatCtx);
    return NULL;
}


int transformVideos(const char **input_files, int num_input_files, const char *output_file, const char *filters_str) {
    AVFormatContext *ic_array[num_input_files];
    AVFormatContext *oc = NULL;
    AVCodecContext *dec_vctx[num_input_files];
    AVCodecContext *en_vctx = NULL;
    
    AVFilterGraph *filter_graph = NULL;
    int ret = 0;

    memset(&ic_array[0],0,num_input_files * sizeof(AVFormatContext *));
    memset(&dec_vctx[0],0,num_input_files * sizeof(AVCodecContext *));
    // Open input files and create array of input format contexts
    for (int i = 0; i < num_input_files; i++) {
        if (avformat_open_input(&ic_array[i], input_files[i], NULL, NULL) != 0) {
            fprintf(stderr, "Could not open input file '%s'\n", input_files[i]);
            goto end;
        }

        // Read stream information for each input file
        if (avformat_find_stream_info(ic_array[i], NULL) < 0) {
            fprintf(stderr, "Could not read stream info for input file '%s'\n", input_files[i]);
            goto end;
        }
        print_stream_info(ic_array[i],0);
        for(int j = 0; j < ic_array[i]->nb_streams; j++) {
            if(ic_array[i]->streams[j]->codecpar->codec_type != AVMEDIA_TYPE_VIDEO)
                continue;
            dec_vctx[i] = create_codec_context(ic_array[i]->streams[j]->codecpar,ic_array[i]->streams[j]->time_base,false);
            if(!dec_vctx[i]) {
                printf("input stream %d codec init fail\n",i);
                goto end;
            }
            break;
        }
        
    }
    // Create output format context and open output file
    AVRational tb = (AVRational){ 1, 15360 };
    AVCodecParameters *v_param = create_video_codec_params(1280,720, 2106102 ,(AVRational){ 30, 1 },AV_PIX_FMT_YUV420P,(AVRational){ 9, 16 });
    oc = create_oc(output_file,tb,v_param,NULL);
    if (!oc) {
        fprintf(stderr, "Could not create output context\n");
        goto end;
    }

    en_vctx = create_codec_context(v_param,tb,true);
    if(!en_vctx) {
        printf("output stream codec init fail\n");
        goto end;
    }
    
    // Write output file header
    if (avformat_write_header(oc, NULL) < 0) {
        fprintf(stderr, "Error occurred when writing header\n");
        goto end;
    }

    // Initialize filter graph
    filter_graph = avfilter_graph_alloc();
    if (!filter_graph) {
        fprintf(stderr, "Unable to create filter graph\n");
        goto end;
    }
    //for video filter only
    ret = create_vfilters_map(filter_graph,num_input_files,&dec_vctx[0],filters_str);
    //there must be at least 3 filters including num_input_files's buffer src and 1 sink
    if (ret<0 || filter_graph->nb_filters < num_input_files+1){
        fprintf(stderr, "Cannot create filter map\n");
        goto end;
    }
    
    /*for(int i=0; i<filter_graph->nb_filters ;i++) {
        AVFilterContext *filter_ctx = filter_graph->filters[i];
        printf("[%s]name[%s]nb_inputs[%d]nb_outputs[%d]---",
            filter_ctx->name,
            filter_ctx->filter->name,
            filter_ctx->filter->nb_inputs,
            filter_ctx->filter->nb_outputs);
        for(int j=0;j<filter_ctx->nb_inputs;j++) {
            printf("src:[%s]name[%s]nb_inputs[%d]nb_outputs[%d]---",
                filter_ctx->inputs[j]->src->name,
                filter_ctx->inputs[j]->src->filter->name,
                filter_ctx->inputs[j]->src->filter->nb_inputs,
                filter_ctx->inputs[j]->src->filter->nb_outputs);
        }
        for(int j=0;j<filter_ctx->nb_outputs;j++) {
            printf("dst:[%s]name[%s]nb_inputs[%d]nb_outputs[%d]---",
                filter_ctx->outputs[j]->dst->name,
                filter_ctx->outputs[j]->dst->filter->name,
                filter_ctx->outputs[j]->dst->filter->nb_inputs,
                filter_ctx->outputs[j]->dst->filter->nb_outputs);
        }

        printf("\n");
    }*/
    
    char *disp = avfilter_graph_dump(filter_graph,NULL);
    if(disp) {
        printf("%s\n",disp);
        av_free(disp);
    }
    
    // Loop through input files and process frames
    for (int i = 0; i < num_input_files; i++) {
        TRANSFORM_INFO info;
        int64_t ts_offset = 0;
        //char name[64];
        //snprintf(name,sizeof(name),"buffer_%d",i);
        info.src_ctx = filter_graph->filters[i];
        info.sink_ctx = filter_graph->filters[filter_graph->nb_filters-1];
        if(!info.src_ctx || !info.sink_ctx) {
            fprintf(stderr, "Cannot find filter buffer src or sink\n");
            goto end;
        }
        info.ic = ic_array[i];
        info.oc = oc;
        info.dec_ctx = dec_vctx[i];
        info.en_ctx = en_vctx;
        printf("packet start transforming for input%d[%s]\n",i,input_files[i]);
        if(ret = trans_vpacket(&info,ts_offset) < 0){
            fprintf(stderr, "Error occurred when transforming\n");
            goto end;
        }
        ts_offset += info.ic->duration;
    }

    // Write trailer to output file
    if(av_write_trailer(oc)<0) {
        fprintf(stderr, "Error occurred when writing trailer\n");
        goto end;
    }
    
end:
    // Clean up and release resources
    printf("============================================\n");
    for (int i = 0; i < num_input_files; i++) {
        if(ic_array[i]) {
            avformat_close_input(&ic_array[i]);
        }
        if(dec_vctx[i]) {
            avcodec_free_context(&dec_vctx[i]);
        }
    }

    if (oc && !(oc->oformat->flags & AVFMT_NOFILE)) {
        avio_closep(&oc->pb);
    }
   
    if(oc) {
        avformat_free_context(oc);
    }
    if(en_vctx)
        avcodec_free_context(&en_vctx);
   
    if(filter_graph) {
        avfilter_graph_free(&filter_graph);
    }
    printf("============================================\n");
    return ret;
}

void writeFrame(FILE *f,AVFrame *frame) {
    for(int i =0; i< frame->height ; i++) {
        fwrite(frame->data[0]+i*frame->linesize[0],frame->width,1,f);
    }
    for(int i =0; i< frame->height/2 ; i++) {
        fwrite(frame->data[1]+i*frame->linesize[1],frame->width/2,1,f);
    }
    for(int i =0; i< frame->height/2 ; i++) {
        fwrite(frame->data[2]+i*frame->linesize[2],frame->width/2,1,f);
    }
}


int trans_vpacket(TRANSFORM_INFO *info,int64_t time_offset) {
    AVPacket *pkt = av_packet_alloc();
    AVPacket *filtered_pkt = av_packet_alloc();
    AVFrame *frame = av_frame_alloc();
    AVFrame *filtered_frame = av_frame_alloc();
    
    int ret = -1;
 
    while (av_read_frame(info->ic, pkt) >= 0) {
        int in_stream_id = pkt->stream_index;
        int out_stream_id = get_stream_index(info->oc,AVMEDIA_TYPE_VIDEO);
        AVStream *istream = info->ic->streams[in_stream_id];
        AVStream *ostream = info->oc->streams[out_stream_id];
        //printf("read pakcet from stream[%d] to stream[%d]\n",in_stream_id,out_stream_id);
        if (out_stream_id>=0 && istream->codecpar->codec_type == AVMEDIA_TYPE_VIDEO) {
            //send packet for decoding
            /*printf("src packet:timebase{%d,%d} pts[%d] dts[%d] duration[%d]\n",
                    istream->time_base.den,
                    istream->time_base.num,
                    pkt->pts,
                    pkt->dts,
                    pkt->duration);*/
            
            ret = avcodec_send_packet(info->dec_ctx, pkt);
            if (ret < 0) {
                fprintf(stderr, "Error submitting a packet for decoding (%s)\n", av_err2str(ret));
                goto end;
            }
            while(1) {
                //receive the decoded data
                ret = avcodec_receive_frame(info->dec_ctx, frame);
                if (ret == AVERROR(EAGAIN) || ret == AVERROR_EOF) {
                    break;
                } else if (ret < 0) {
                    fprintf(stderr, "Error during decoding (%s)\n", av_err2str(ret));
                    goto end;
                }
                /*printf("decoded frame:pktdts[%d] timebase{%d,%d} pts[%d] bstpst[%d] sample_rate[%d]\n",
                    frame->pkt_dts,
                    frame->time_base.den,
                    frame->time_base.num,
                    frame->pts,
                    frame->best_effort_timestamp,
                    frame->sample_rate);*/
                frame->pts = frame->best_effort_timestamp + time_offset;
                frame->pkt_dts += time_offset;
                //发送到filter
                ret = av_buffersrc_write_frame(info->src_ctx, frame);
                //ret = av_buffersrc_add_frame_flags(info->src_ctx, frame, AV_BUFFERSRC_FLAG_KEEP_REF);
                if (ret < 0) {
                    fprintf(stderr, "Error while writing frame to buffer source\n");
                    goto end;
                }
                av_frame_unref(frame);
            }
            av_packet_unref(pkt); 
            while(1) {
                //从flter读取
                ret = av_buffersink_get_frame(info->sink_ctx, filtered_frame);
                if (ret == AVERROR(EAGAIN) || ret == AVERROR_EOF){
                    break;
                }
                if (ret < 0) {
                    fprintf(stderr, "Error while getting filtered frame from buffer sink\n");
                    goto end;
                }
                /*printf("filtered frame:pktdts[%d] timebase{%d,%d} pts[%d] sample_rate[%d]\n",
                    filtered_frame->pkt_dts,
                    filtered_frame->time_base.den,
                    filtered_frame->time_base.num,
                    filtered_frame->pts,
                    filtered_frame->sample_rate);*/
                filtered_frame->pts = filtered_frame->pkt_dts;
                // 编码AVFrame到AVPacket
                ret = avcodec_send_frame(info->en_ctx, filtered_frame);
                if (ret < 0) {
                    fprintf(stderr, "Error while send filtered frame:%s\n",av_err2str(ret));
                    goto end;
                }
                av_frame_unref(filtered_frame);
            }

            //接受经过filter过的包
            while(1) {
                ret = avcodec_receive_packet(info->en_ctx, filtered_pkt);
                if (ret == AVERROR(EAGAIN) || ret == AVERROR_EOF){
                    break;
                }
                if (ret < 0) {
                    fprintf(stderr, "Error while receiving filtered packet\n");
                    goto end;
                }
                /*printf("filtered packet:timebase{%d,%d} pts[%d] dts[%d] duration[%d]\n",
                    istream->time_base.den,
                    istream->time_base.num,
                    filtered_pkt->pts,
                    filtered_pkt->dts,
                    filtered_pkt->duration);
                printf("out timebase{%d,%d} offset[%d]\n",ostream->time_base.den,ostream->time_base.num,time_offset);*/
                // rescale output packet timestamp values from codec to stream timebase
                av_packet_rescale_ts(filtered_pkt, istream->time_base, ostream->time_base);
                filtered_pkt->stream_index = out_stream_id;
                ret = av_interleaved_write_frame(info->oc, filtered_pkt);
                if (ret < 0) {
                    fprintf(stderr, "Error while writing filtered packet to output file\n");
                    goto end;
                }
                av_packet_unref(filtered_pkt); 
            }
        }
        else {
            //av_packet_rescale_ts(&packet, codec_ctx->time_base, output_stream->time_base);
            /*ret = av_interleaved_write_frame(oc, pkt);
            if (ret < 0) {
                fprintf(stderr, "Error while writing packet to output file\n");
                break;
            }*/
        }
    }
    
    
end:
    av_packet_free(&pkt);
    av_frame_free(&filtered_frame);
    av_frame_free(&frame);
    if (ret == AVERROR(EAGAIN) || ret == AVERROR_EOF)
        return 0;
    return ret;
}



int transferSubtitles(FILE *subtitle_file,AVFormatContext *oc, AVStream *subtitle_stream) {
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
                //AVStream *subtitle_stream = oc->streams[packet.stream_index];
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
                av_interleaved_write_frame(oc, &packet);
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
